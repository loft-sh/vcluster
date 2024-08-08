package pods

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/resources/pods/token"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	translatepods "github.com/loft-sh/vcluster/pkg/controllers/resources/pods/translate"
	"github.com/loft-sh/vcluster/pkg/util/toleration"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	// Default grace period in seconds
	minimumGracePeriodInSeconds int64 = 30
	zero                              = int64(0)
)

func New(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	virtualClusterClient, err := kubernetes.NewForConfig(ctx.VirtualManager.GetConfig())
	if err != nil {
		return nil, err
	}
	physicalClusterClient, err := kubernetes.NewForConfig(ctx.PhysicalManager.GetConfig())
	if err != nil {
		return nil, err
	}

	// parse node selector
	var nodeSelector *metav1.LabelSelector
	if len(ctx.Config.Sync.FromHost.Nodes.Selector.Labels) > 0 {
		nodeSelector = &metav1.LabelSelector{
			MatchLabels: ctx.Config.Sync.FromHost.Nodes.Selector.Labels,
		}
	}

	// parse tolerations
	var tolerations []*corev1.Toleration
	if len(ctx.Config.Sync.ToHost.Pods.EnforceTolerations) > 0 {
		for _, t := range ctx.Config.Sync.ToHost.Pods.EnforceTolerations {
			tol, err := toleration.ParseToleration(t)
			if err == nil {
				tolerations = append(tolerations, &tol)
			}
		}
	}

	// get pods mapper
	podsMapper, err := ctx.Mappings.ByGVK(mappings.Pods())
	if err != nil {
		return nil, err
	}

	// create new namespaced translator
	genericTranslator := translator.NewGenericTranslator(ctx, "pod", &corev1.Pod{}, podsMapper)

	// create pod translator
	podTranslator, err := translatepods.NewTranslator(ctx, genericTranslator.EventRecorder())
	if err != nil {
		return nil, errors.Wrap(err, "create pod translator")
	}

	return &podSyncer{
		GenericTranslator: genericTranslator,
		Importer:          pro.NewImporter(podsMapper),

		serviceName:     ctx.Config.WorkloadService,
		enableScheduler: ctx.Config.ControlPlane.Advanced.VirtualScheduler.Enabled,

		virtualClusterClient:  virtualClusterClient,
		physicalClusterClient: physicalClusterClient,
		physicalClusterConfig: ctx.PhysicalManager.GetConfig(),
		podTranslator:         podTranslator,
		nodeSelector:          nodeSelector,
		tolerations:           tolerations,

		podSecurityStandard: ctx.Config.Policies.PodSecurityStandard,
	}, nil
}

type podSyncer struct {
	syncertypes.GenericTranslator
	syncertypes.Importer

	serviceName     string
	enableScheduler bool

	podTranslator         translatepods.Translator
	virtualClusterClient  kubernetes.Interface
	physicalClusterClient kubernetes.Interface
	physicalClusterConfig *rest.Config
	nodeSelector          *metav1.LabelSelector
	tolerations           []*corev1.Toleration

	podSecurityStandard string
}

var _ syncertypes.ControllerModifier = &podSyncer{}

func (s *podSyncer) ModifyController(registerContext *synccontext.RegisterContext, builder *builder.Builder) (*builder.Builder, error) {
	eventHandler := handler.Funcs{
		UpdateFunc: func(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
			// no need to reconcile pods if namespace labels didn't change
			if reflect.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels()) {
				return
			}

			ns := e.ObjectNew.GetName()
			pods := &corev1.PodList{}
			err := registerContext.VirtualManager.GetClient().List(ctx, pods, client.InNamespace(ns))
			if err != nil {
				klog.FromContext(ctx).Info("failed to list pods in the namespace when handling namespace update", "namespace", ns, "error", err)
				return
			}
			for _, pod := range pods.Items {
				q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      pod.GetName(),
					Namespace: ns,
				}})
			}
		},
	}

	return builder.Watches(&corev1.Namespace{}, eventHandler), nil
}

var _ syncertypes.Syncer = &podSyncer{}

func (s *podSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*corev1.Pod](s)
}

func (s *podSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*corev1.Pod]) (ctrl.Result, error) {
	// in some scenarios it is possible that the pod was already started and the physical pod
	// was deleted without vcluster's knowledge. In this case we are deleting the virtual pod
	// as well, to avoid conflicts with nodes if we would resync the same pod to the host cluster again.
	if event.IsDelete() || event.Virtual.DeletionTimestamp != nil || event.Virtual.Status.StartTime != nil {
		// delete pod immediately
		ctx.Log.Infof("delete pod %s/%s immediately, because it is being deleted & there is no physical pod", event.Virtual.Namespace, event.Virtual.Name)
		err := ctx.VirtualClient.Delete(ctx, event.Virtual, &client.DeleteOptions{
			GracePeriodSeconds: &zero,
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// validate virtual pod before syncing it to the host cluster
	if s.podSecurityStandard != "" {
		valid, err := s.isPodSecurityStandardsValid(ctx, event.Virtual, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		} else if !valid {
			return ctrl.Result{}, nil
		}
	}

	// translate the pod
	pPod, err := s.translate(ctx, event.Virtual)
	if err != nil {
		return ctrl.Result{}, err
	}

	// ensure tolerations
	for _, tol := range s.tolerations {
		pPod.Spec.Tolerations = append(pPod.Spec.Tolerations, *tol)
	}

	// ensure node selector
	if s.nodeSelector != nil {
		// 2 cases:
		// 1. Pod already has a nodeName -> then we check if the node exists in the virtual cluster
		// 2. Pod has no nodeName -> then we set the nodeSelector
		if pPod.Spec.NodeName == "" {
			if pPod.Spec.NodeSelector == nil {
				pPod.Spec.NodeSelector = map[string]string{}
			}
			for k, v := range s.nodeSelector.MatchLabels {
				pPod.Spec.NodeSelector[k] = v
			}
		} else {
			// make sure the node does exist in the virtual cluster
			err = ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: pPod.Spec.NodeName}, &corev1.Node{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}

				s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncWarning", "Given nodeName %s does not exist in virtual cluster", pPod.Spec.NodeName)
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
		}
	}

	// if scheduler is enabled we only sync if the pod has a node name
	if s.enableScheduler && pPod.Spec.NodeName == "" {
		return ctrl.Result{}, nil
	}

	err = pro.ApplyPatchesHostObject(ctx, nil, pPod, event.Virtual, ctx.Config.Sync.ToHost.Pods.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateHostObject(ctx, event.Virtual, pPod, s.EventRecorder())
}

func (s *podSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*corev1.Pod]) (_ ctrl.Result, retErr error) {
	// should pod get deleted?
	if event.Host.DeletionTimestamp != nil {
		if event.Virtual.DeletionTimestamp == nil {
			gracePeriod := minimumGracePeriodInSeconds
			if event.Virtual.Spec.TerminationGracePeriodSeconds != nil {
				gracePeriod = *event.Virtual.Spec.TerminationGracePeriodSeconds
			}

			ctx.Log.Infof("delete virtual pod %s/%s, because the physical pod is being deleted", event.Virtual.Namespace, event.Virtual.Name)
			if err := ctx.VirtualClient.Delete(ctx, event.Virtual, &client.DeleteOptions{GracePeriodSeconds: &gracePeriod}); err != nil {
				return ctrl.Result{}, err
			}
		} else if *event.Virtual.DeletionGracePeriodSeconds != *event.Host.DeletionGracePeriodSeconds {
			ctx.Log.Infof("delete virtual pPod %s/%s with grace period seconds %v", event.Virtual.Namespace, event.Virtual.Name, *event.Host.DeletionGracePeriodSeconds)
			if err := ctx.VirtualClient.Delete(ctx, event.Virtual, &client.DeleteOptions{GracePeriodSeconds: event.Host.DeletionGracePeriodSeconds, Preconditions: metav1.NewUIDPreconditions(string(event.Virtual.UID))}); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	} else if event.Virtual.DeletionTimestamp != nil {
		ctx.Log.Infof("delete physical pod %s/%s, because virtual pod is being deleted", event.Host.Namespace, event.Host.Name)
		err := ctx.PhysicalClient.Delete(ctx, event.Host, &client.DeleteOptions{
			GracePeriodSeconds: event.Virtual.DeletionGracePeriodSeconds,
			Preconditions:      metav1.NewUIDPreconditions(string(event.Host.UID)),
		})
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// make sure node exists for pod
	if event.Host.Spec.NodeName != "" {
		requeue, err := s.ensureNode(ctx, event.Host, event.Virtual)
		if kerrors.IsConflict(err) {
			ctx.Log.Debugf("conflict binding virtual pod %s/%s", event.Virtual.Namespace, event.Virtual.Name)
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		} else if requeue {
			return ctrl.Result{Requeue: true}, nil
		}
	} else if event.Host.Spec.NodeName != "" && event.Virtual.Spec.NodeName != "" && event.Host.Spec.NodeName != event.Virtual.Spec.NodeName {
		// if physical pod nodeName is different from virtual pod nodeName, we delete the virtual one
		ctx.Log.Infof("delete virtual pod %s/%s, because node name is different between the two", event.Virtual.Namespace, event.Virtual.Name)
		err := ctx.VirtualClient.Delete(ctx, event.Virtual, &client.DeleteOptions{GracePeriodSeconds: &minimumGracePeriodInSeconds})
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// validate virtual pod before syncing it to the host cluster
	if s.podSecurityStandard != "" {
		valid, err := s.isPodSecurityStandardsValid(ctx, event.Virtual, ctx.Log)
		if err != nil {
			return ctrl.Result{}, err
		} else if !valid {
			return ctrl.Result{}, nil
		}
	}

	// sync ephemeral containers
	if syncEphemeralContainers(event.Virtual, event.Host) {
		kubeIP, _, ptrServiceList, err := s.getK8sIPDNSIPServiceList(ctx, event.Virtual)
		if err != nil {
			return ctrl.Result{}, err
		}

		// translate services to environment variables
		serviceEnv := translatepods.ServicesToEnvironmentVariables(event.Virtual.Spec.EnableServiceLinks, ptrServiceList, kubeIP)
		for i := range event.Virtual.Spec.EphemeralContainers {
			envVar, envFrom, err := s.podTranslator.TranslateContainerEnv(ctx, event.Virtual.Spec.EphemeralContainers[i].Env, event.Virtual.Spec.EphemeralContainers[i].EnvFrom, event.Virtual, serviceEnv)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("translate container env: %w", err)
			}
			event.Virtual.Spec.EphemeralContainers[i].Env = envVar
			event.Virtual.Spec.EphemeralContainers[i].EnvFrom = envFrom
		}

		// add ephemeralContainers subresource to physical pod
		err = AddEphemeralContainer(ctx, s.physicalClusterClient, event.Host, event.Virtual)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// set pod owner as sa token
	err := setSATokenSecretAsOwner(ctx, ctx.PhysicalClient, event.Virtual, event.Host)
	if err != nil {
		return ctrl.Result{}, err
	}

	// patch objects
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.ToHost.Pods.Translate))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}

		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// update the virtual pod if the spec has changed
	err = s.podTranslator.Diff(ctx, event)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (s *podSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*corev1.Pod]) (_ ctrl.Result, retErr error) {
	if event.IsDelete() || event.Host.DeletionTimestamp != nil {
		// virtual object is not here anymore, so we delete
		return syncer.DeleteHostObject(ctx, event.Host, "virtual object was deleted")
	}

	vPod := translate.VirtualMetadata(event.Host, s.HostToVirtual(ctx, types.NamespacedName{Name: event.Host.GetName(), Namespace: event.Host.GetNamespace()}, event.Host))
	vPod.Spec.NodeName = ""
	if !ctx.Config.Sync.ToHost.ServiceAccounts.Enabled {
		vPod.Spec.ServiceAccountName = ""
		vPod.Spec.DeprecatedServiceAccount = ""
	}

	err := pro.ApplyPatchesVirtualObject(ctx, nil, vPod, event.Host, ctx.Config.Sync.ToHost.Pods.Translate)
	if err != nil {
		return ctrl.Result{}, err
	}

	return syncer.CreateVirtualObject(ctx, event.Host, vPod, s.EventRecorder())
}

func setSATokenSecretAsOwner(ctx *synccontext.SyncContext, pClient client.Client, vObj, pObj *corev1.Pod) error {
	if !ctx.Config.Sync.ToHost.Pods.UseSecretsForSATokens {
		return nil
	}

	secret, err := token.GetSecretIfExists(ctx, pClient, vObj.Name, vObj.Namespace)
	if err := token.IgnoreAcceptableErrors(err); err != nil {
		return err
	} else if secret != nil {
		// check if owner is vCluster service, if so, modify to pod as owner
		err := token.SetPodAsOwner(ctx, pObj, pClient, secret)
		if err != nil {
			return err
		}
	}

	return nil
}

func syncEphemeralContainers(vPod *corev1.Pod, pPod *corev1.Pod) bool {
	if vPod.Spec.EphemeralContainers == nil {
		return false
	}
	if len(vPod.Spec.EphemeralContainers) != len(pPod.Spec.EphemeralContainers) {
		return true
	}
	for i := range vPod.Spec.EphemeralContainers {
		if vPod.Spec.EphemeralContainers[i].Image != pPod.Spec.EphemeralContainers[i].Image {
			return true
		}
		if vPod.Spec.EphemeralContainers[i].Name != pPod.Spec.EphemeralContainers[i].Name {
			return true
		}
	}
	return false
}

func (s *podSyncer) ensureNode(ctx *synccontext.SyncContext, pObj *corev1.Pod, vObj *corev1.Pod) (bool, error) {
	if vObj.Spec.NodeName != pObj.Spec.NodeName && vObj.Spec.NodeName != "" {
		// node of virtual and physical pod are different, we delete the virtual pod to try to recover from this state
		ctx.Log.Infof("delete virtual pod %s/%s, because virtual and physical pods have different assigned nodes", vObj.Namespace, vObj.Name)
		err := ctx.VirtualClient.Delete(ctx, vObj)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	// ensure the node is available in the virtual cluster, if not and we sync the pod to the virtual cluster,
	// it will get deleted automatically by kubernetes so we ensure the node is synced
	vNode := &corev1.Node{}
	err := ctx.VirtualClient.Get(ctx, types.NamespacedName{Name: pObj.Spec.NodeName}, vNode)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			ctx.Log.Infof("error retrieving virtual node %s: %v", pObj.Spec.NodeName, err)
			return false, err
		}

		return true, nil
	}

	if vObj.Spec.NodeName != pObj.Spec.NodeName {
		err = s.assignNodeToPod(ctx, pObj, vObj)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (s *podSyncer) assignNodeToPod(ctx *synccontext.SyncContext, pObj *corev1.Pod, vObj *corev1.Pod) error {
	ctx.Log.Infof("bind virtual pod %s/%s to node %s, because node name between physical and virtual is different", vObj.Namespace, vObj.Name, pObj.Spec.NodeName)
	err := s.virtualClusterClient.CoreV1().Pods(vObj.Namespace).Bind(ctx, &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vObj.Name,
			Namespace: vObj.Namespace,
		},
		Target: corev1.ObjectReference{
			Kind:       "Node",
			Name:       pObj.Spec.NodeName,
			APIVersion: "v1",
		},
	}, metav1.CreateOptions{})
	if err != nil {
		if !kerrors.IsConflict(err) {
			s.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error binding pod: %v", err)
		}
		return err
	}

	// wait until cache is updated
	err = wait.PollUntilContextTimeout(ctx, time.Millisecond*50, time.Second*2, true, func(syncContext context.Context) (done bool, err error) {
		vPod := &corev1.Pod{}
		err = ctx.VirtualClient.Get(syncContext, types.NamespacedName{Namespace: vObj.Namespace, Name: vObj.Name}, vPod)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return vPod.Spec.NodeName != "", nil
	})
	return err
}

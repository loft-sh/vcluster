package namespaces

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Unsafe annotations based on the docs here:
// https://kubernetes.io/docs/reference/labels-annotations-taints/
var excludedAnnotations = []string{
	"scheduler.alpha.kubernetes.io/node-selector",
	"scheduler.alpha.kubernetes.io/defaultTolerations",
}

const (
	VClusterNameAnnotation      = "vcluster.loft.sh/vcluster-name"
	VClusterNamespaceAnnotation = "vcluster.loft.sh/vcluster-namespace"
)

func New(ctx *synccontext.RegisterContext) (types.Object, error) {
	mapper, err := ctx.Mappings.ByGVK(mappings.Namespaces())
	if err != nil {
		return nil, err
	}

	namespaceLabels := map[string]string{}
	for k, v := range ctx.Config.Experimental.MultiNamespaceMode.NamespaceLabels {
		namespaceLabels[k] = v
	}
	namespaceLabels[VClusterNameAnnotation] = ctx.Config.Name
	namespaceLabels[VClusterNamespaceAnnotation] = ctx.CurrentNamespace

	return &namespaceSyncer{
		GenericTranslator:          translator.NewGenericTranslator(ctx, "namespace", &corev1.Namespace{}, mapper),
		workloadServiceAccountName: ctx.Config.ControlPlane.Advanced.WorkloadServiceAccount.Name,

		excludedAnnotations: excludedAnnotations,

		namespaceLabels: namespaceLabels,
	}, nil
}

type namespaceSyncer struct {
	types.GenericTranslator
	workloadServiceAccountName string

	excludedAnnotations []string

	namespaceLabels map[string]string
}

var _ types.Syncer = &namespaceSyncer{}

func (s *namespaceSyncer) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	newNamespace := s.translate(ctx, vObj.(*corev1.Namespace))
	ctx.Log.Infof("create physical namespace %s", newNamespace.Name)
	err := ctx.PhysicalClient.Create(ctx, newNamespace)
	if err != nil {
		ctx.Log.Infof("error syncing %s to physical cluster: %v", vObj.GetName(), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, newNamespace.Name)
}

func (s *namespaceSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, pObj, vObj)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patch.Patch(ctx, pObj, vObj); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	// cast objects
	pNamespace, vNamespace, _, _ := synccontext.Cast[*corev1.Namespace](ctx, pObj, vObj)

	s.translateUpdate(ctx, pNamespace, vNamespace)

	return ctrl.Result{}, s.EnsureWorkloadServiceAccount(ctx, pNamespace.Name)
}

func (s *namespaceSyncer) EnsureWorkloadServiceAccount(ctx *synccontext.SyncContext, pNamespace string) error {
	if s.workloadServiceAccountName == "" {
		return nil
	}

	svc := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pNamespace,
			Name:      s.workloadServiceAccountName,
		},
	}
	_, err := controllerutil.CreateOrPatch(ctx, ctx.PhysicalClient, svc, func() error { return nil })
	return err
}

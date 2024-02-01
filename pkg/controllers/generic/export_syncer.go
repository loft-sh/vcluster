package generic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/pkg/setup/options"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/loft-sh/vcluster/pkg/log"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	patchesregex "github.com/loft-sh/vcluster/pkg/patches/regex"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func CreateExporters(ctx *options.ControllerContext, exporterConfig *config.Config) error {
	if len(exporterConfig.Exports) == 0 {
		return nil
	}

	scheme := ctx.LocalManager.GetScheme()
	registerCtx := util.ToRegisterContext(ctx)

	for _, exportConfig := range exporterConfig.Exports {
		gvk := schema.FromAPIVersionAndKind(exportConfig.APIVersion, exportConfig.Kind)
		if !scheme.Recognizes(gvk) {
			_, _, err := translate.EnsureCRDFromPhysicalCluster(
				registerCtx.Context,
				registerCtx.PhysicalManager.GetConfig(),
				registerCtx.VirtualManager.GetConfig(),
				gvk)
			if err != nil {
				if exportConfig.Optional {
					klog.Infof("error ensuring CRD %s(%s) from host cluster: %v. Skipping exportSyncer as resource is optional", exportConfig.Kind, exportConfig.APIVersion, err)
					continue
				}

				return fmt.Errorf("error creating %s(%s) syncer: %w", exportConfig.Kind, exportConfig.APIVersion, err)
			}
		}

		reversePatches := []*config.Patch{
			{
				Operation: config.PatchTypeCopyFromObject,
				FromPath:  "status",
				Path:      "status",
			},
		}
		reversePatches = append(reversePatches, exportConfig.ReversePatches...)
		exportConfig.ReversePatches = reversePatches

		s, err := createExporter(registerCtx, exportConfig)
		klog.Infof("creating exporter for %s/%s", exportConfig.APIVersion, exportConfig.Kind)
		if err != nil {
			return fmt.Errorf("error creating %s(%s) syncer: %w", exportConfig.Kind, exportConfig.APIVersion, err)
		}

		err = syncer.RegisterSyncer(registerCtx, s)
		klog.Infof("registering export syncer for %s/%s", exportConfig.APIVersion, exportConfig.Kind)
		if err != nil {
			return fmt.Errorf("error registering syncer %w", err)
		}
	}

	return nil
}

func createExporter(ctx *synccontext.RegisterContext, config *config.Export) (syncertypes.Syncer, error) {
	obj := &unstructured.Unstructured{}
	obj.SetKind(config.Kind)
	obj.SetAPIVersion(config.APIVersion)

	err := validateExportConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration for %s(%s) mapping: %w", config.Kind, config.APIVersion, err)
	}

	var selector labels.Selector
	if config.Selector != nil {
		selector, err = metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(config.Selector.LabelSelector))
		if err != nil {
			return nil, fmt.Errorf("invalid selector in configuration for %s(%s) mapping: %w", config.Kind, config.APIVersion, err)
		}
	}

	statusIsSubresource := true
	// TODO: [low priority] check if config.Kind + config.APIVersion has status subresource

	gvk := schema.FromAPIVersionAndKind(config.APIVersion, config.Kind)
	controllerID := fmt.Sprintf("%s/%s/GenericExport", strings.ToLower(gvk.Kind), strings.ToLower(gvk.Group))
	return &exporter{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, controllerID, obj),
		patcher: &patcher{
			fromClient:          ctx.VirtualManager.GetClient(),
			toClient:            ctx.PhysicalManager.GetClient(),
			statusIsSubresource: statusIsSubresource,
			log:                 log.New(controllerID),
		},
		gvk:      gvk,
		config:   config,
		selector: selector,
		name:     controllerID,
	}, nil
}

type exporter struct {
	translator.NamespacedTranslator

	patcher  *patcher
	gvk      schema.GroupVersionKind
	config   *config.Export
	selector labels.Selector
	name     string
}

func (f *exporter) SyncToHost(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// check if selector matches
	if !f.objectMatches(vObj) {
		return ctrl.Result{}, nil
	}

	// apply object to physical cluster
	ctx.Log.Infof("Create physical %s %s/%s, since it is missing, but virtual object exists", f.config.Kind, vObj.GetNamespace(), vObj.GetName())
	pObj, err := f.patcher.ApplyPatches(ctx.Context, vObj, nil, f.config.Patches, f.config.ReversePatches, func(vObj client.Object) (client.Object, error) {
		return f.TranslateMetadata(ctx.Context, vObj), nil
	}, &virtualToHostNameResolver{namespace: vObj.GetNamespace(), targetNamespace: translate.Default.PhysicalNamespace(vObj.GetNamespace())})
	if kerrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	// wait here for vObj to be created
	err = wait.PollUntilContextTimeout(ctx.Context, time.Millisecond*10, time.Second, true, func(pollContext context.Context) (done bool, err error) {
		err = ctx.PhysicalClient.Get(pollContext, types.NamespacedName{
			Namespace: pObj.GetNamespace(),
			Name:      pObj.GetName(),
		}, f.Resource())
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// all good we can return safely
	return ctrl.Result{}, nil
}

func (f *exporter) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	// check if virtual object is not matching anymore
	if !f.objectMatches(vObj) {
		ctx.Log.Infof("delete physical %s %s/%s, because it is not used anymore", f.config.Kind, pObj.GetNamespace(), pObj.GetName())
		err := ctx.PhysicalClient.Delete(ctx.Context, pObj, &client.DeleteOptions{
			GracePeriodSeconds: &[]int64{0}[0],
		})
		if err != nil {
			ctx.Log.Infof("error deleting physical %s %s/%s in physical cluster: %v", f.config.Kind, pObj.GetNamespace(), pObj.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// check if either object is getting deleted
	if vObj.GetDeletionTimestamp() != nil || pObj.GetDeletionTimestamp() != nil {
		if pObj.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete physical object %s/%s, because the virtual object is being deleted", pObj.GetNamespace(), pObj.GetName())
			if err := ctx.PhysicalClient.Delete(ctx.Context, pObj); err != nil {
				return ctrl.Result{}, err
			}
		} else if vObj.GetDeletionTimestamp() == nil {
			ctx.Log.Infof("delete virtual object %s/%s, because physical object is being deleted", vObj.GetNamespace(), vObj.GetName())
			if err := ctx.VirtualClient.Delete(ctx.Context, vObj); err != nil {
				return ctrl.Result{}, nil
			}
		}

		return ctrl.Result{}, nil
	}

	// apply reverse patches
	result, err := f.patcher.ApplyReversePatches(ctx.Context, vObj, pObj, f.config.ReversePatches, &hostToVirtualNameResolver{
		gvk:  f.gvk,
		pObj: pObj,
	})
	if err != nil {
		if kerrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		if kerrors.IsInvalid(err) {
			ctx.Log.Infof("Warning: this message could indicate a timing issue with no significant impact, or a bug. Please report this if your resource never reaches the expected state. Error message: failed to patch virtual %s %s/%s: %v", f.config.Kind, vObj.GetNamespace(), vObj.GetName(), err)
			// this happens when some field is being removed shortly after being added, which suggest it's a timing issue
			// it doesn't seem to have any negative consequence besides the logged error message
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("failed to patch virtual %s %s/%s: %w", f.config.Kind, vObj.GetNamespace(), vObj.GetName(), err)
	} else if result == controllerutil.OperationResultUpdated || result == controllerutil.OperationResultUpdatedStatus || result == controllerutil.OperationResultUpdatedStatusOnly {
		// a change will trigger reconciliation anyway, and at that point we can make
		// a more accurate updates(reverse patches) to the virtual resource
		return ctrl.Result{}, nil
	}

	// apply patches
	_, err = f.patcher.ApplyPatches(ctx.Context, vObj, pObj, f.config.Patches, f.config.ReversePatches, func(vObj client.Object) (client.Object, error) {
		return f.TranslateMetadata(ctx.Context, vObj), nil
	}, &virtualToHostNameResolver{
		namespace:       vObj.GetNamespace(),
		targetNamespace: translate.Default.PhysicalNamespace(vObj.GetNamespace())})
	if err != nil {
		// when invalid, auto delete and recreate to recover
		if kerrors.IsInvalid(err) && f.config.ReplaceWhenInvalid {
			// Replace the object
			ctx.Log.Infof("Replace physical object, because apply failed: %v", err)
			err = ctx.PhysicalClient.Delete(ctx.Context, pObj, &client.DeleteOptions{
				GracePeriodSeconds: &[]int64{0}[0],
			})
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		if kerrors.IsConflict(err) {
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	return ctrl.Result{}, nil
}

var _ syncertypes.ToVirtualSyncer = &exporter{}

func (f *exporter) SyncToVirtual(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	if !translate.Default.IsManaged(pObj) {
		return ctrl.Result{}, nil
	}

	// delete physical object because virtual one is missing
	return syncer.DeleteObject(ctx, pObj, fmt.Sprintf("delete physical %s because virtual is missing", pObj.GetName()))
}

func (f *exporter) Name() string {
	return f.name
}

// TranslateMetadata converts the virtual object into a physical object
func (f *exporter) TranslateMetadata(ctx context.Context, vObj client.Object) client.Object {
	pObj := f.NamespacedTranslator.TranslateMetadata(ctx, vObj)
	if pObj.GetAnnotations() == nil {
		pObj.SetAnnotations(map[string]string{translate.ControllerLabel: f.Name()})
	} else {
		a := pObj.GetAnnotations()
		a[translate.ControllerLabel] = f.Name()
		pObj.SetAnnotations(a)
	}
	return pObj
}

func (f *exporter) IsManaged(_ context.Context, pObj client.Object) (bool, error) {
	return translate.Default.IsManaged(pObj), nil
}

func (f *exporter) objectMatches(obj client.Object) bool {
	return f.selector == nil || f.selector.Matches(labels.Set(obj.GetLabels()))
}

type virtualToHostNameResolver struct {
	namespace       string
	targetNamespace string
}

func (r *virtualToHostNameResolver) TranslateName(name string, regex *regexp.Regexp, _ string) (string, error) {
	return r.TranslateNameWithNamespace(name, r.namespace, regex, "")
}

func (r *virtualToHostNameResolver) TranslateNameWithNamespace(name string, namespace string, regex *regexp.Regexp, _ string) (string, error) {
	if regex != nil {
		return patchesregex.ProcessRegex(regex, name, func(name, ns string) types.NamespacedName {
			// if the regex match doesn't contain namespace - use the namespace set in this resolver
			if ns == "" {
				ns = namespace
			}

			return types.NamespacedName{
				Namespace: translate.Default.PhysicalNamespace(namespace),
				Name:      translate.Default.PhysicalName(name, ns)}
		}), nil
	}

	return translate.Default.PhysicalName(name, namespace), nil
}

func (r *virtualToHostNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return translate.Default.TranslateLabelSelectorCluster(selector), nil
}

func (r *virtualToHostNameResolver) TranslateLabelKey(key string) (string, error) {
	return translate.Default.ConvertLabelKey(key), nil
}

func (r *virtualToHostNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: selector,
	}

	return metav1.LabelSelectorAsMap(
		translate.Default.TranslateLabelSelector(labelSelector))
}

func (r *virtualToHostNameResolver) TranslateNamespaceRef(namespace string) (string, error) {
	return translate.Default.PhysicalNamespace(namespace), nil
}

func validateExportConfig(config *config.Export) error {
	for _, p := range append(config.Patches, config.ReversePatches...) {
		if p.Regex != "" {
			parsed, err := patchesregex.PrepareRegex(p.Regex)
			if err != nil {
				return fmt.Errorf("invalid Regex: %w", err)
			}
			p.ParsedRegex = parsed
		}
	}
	return nil
}

type hostToVirtualNameResolver struct {
	gvk  schema.GroupVersionKind
	pObj client.Object
}

func (r *hostToVirtualNameResolver) TranslateName(string, *regexp.Regexp, string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateNameWithNamespace(string, string, *regexp.Regexp, string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelKey(string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelExpressionsSelector(*metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelSelector(map[string]string) (map[string]string, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateNamespaceRef(string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}

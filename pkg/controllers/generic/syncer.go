package generic

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/log"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	patchesregex "github.com/loft-sh/vcluster/pkg/patches/regex"
	util "github.com/loft-sh/vcluster/pkg/util/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	VclusterTranslationObjectNameKey = "vcluster.loft.sh/object-name"
)

func CreateExporters(ctx *context.ControllerContext, config *config.Config) error {
	scheme := ctx.LocalManager.GetScheme()
	registerCtx := util.ToRegisterContext(ctx)

	for _, exportConfig := range config.Exports {
		gvk := schema.FromAPIVersionAndKind(exportConfig.APIVersion, exportConfig.Kind)
		if !scheme.Recognizes(gvk) {
			err := translate.EnsureCRDFromPhysicalCluster(
				registerCtx.Context,
				registerCtx.PhysicalManager.GetConfig(),
				registerCtx.VirtualManager.GetConfig(),
				gvk)
			if err != nil {
				klog.Errorf("Error syncronizing CRD %s(%s) from the host cluster into vcluster: %v", exportConfig.Kind, exportConfig.APIVersion, err)
				return err
			}
		}
	}

	for _, exportConfig := range config.Exports {
		s, err := createExporter(registerCtx, exportConfig)
		if err != nil {
			klog.Errorf("Error creating %s(%s) syncer: %v", exportConfig.Kind, exportConfig.APIVersion, err)
			return err
		}

		err = syncer.RegisterSyncer(registerCtx, s)
		if err != nil {
			klog.Errorf("Error registering syncer %v", err)
		}
	}

	return nil
}

func createExporter(ctx *synccontext.RegisterContext, config *config.Export) (syncer.Syncer, error) {
	obj := &unstructured.Unstructured{}
	obj.SetKind(config.Kind)
	obj.SetAPIVersion(config.APIVersion)

	err := validateExportConfig(config)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration for %s(%s) mapping: %v", config.Kind, config.APIVersion, err)
	}

	var selector labels.Selector
	if config.Selector != nil {
		selector, err = metav1.LabelSelectorAsSelector(metav1.SetAsLabelSelector(config.Selector.LabelSelector))
		if err != nil {
			return nil, fmt.Errorf("invalid selector in configuration for %s(%s) mapping: %v", config.Kind, config.APIVersion, err)
		}
	}

	statusIsSubresource := true
	// TODO: [low priority] check if config.Kind + config.APIVersion has status subresource

	return &exporter{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, config.Kind+"-exporter", obj),
		patcher: &patcher{
			fromClient:          ctx.VirtualManager.GetClient(),
			toClient:            ctx.PhysicalManager.GetClient(),
			statusIsSubresource: statusIsSubresource,
			log:                 log.New(config.Kind + "-exporter"),
		},
		gvk:      schema.FromAPIVersionAndKind(config.APIVersion, config.Kind),
		config:   config,
		selector: selector,
	}, nil
}

type exporter struct {
	translator.NamespacedTranslator

	patcher *patcher
	gvk     schema.GroupVersionKind

	config *config.Export

	selector labels.Selector
}

func (f *exporter) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// check if selector matches
	if isControlled(vObj) || !f.objectMatches(vObj) {
		return ctrl.Result{}, nil
	}

	// apply object to physical cluster
	ctx.Log.Infof("Create physical %s %s/%s, since it is missing, but virtual object exists", f.config.Kind, vObj.GetNamespace(), vObj.GetName())
	_, err := f.patcher.ApplyPatches(ctx.Context, vObj, nil, f.config.Patches, f.config.ReversePatches, func(vObj client.Object) (client.Object, error) {
		return f.TranslateMetadata(vObj), nil
	}, &virtualToHostNameResolver{namespace: vObj.GetNamespace(), targetNamespace: translate.Default.PhysicalNamespace(vObj.GetNamespace())})
	if err != nil {
		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %v", err)
	}

	return ctrl.Result{}, nil
}
func (f *exporter) isExcluded(pObj client.Object) bool {
	labels := pObj.GetLabels()
	return labels == nil
}

func (f *exporter) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	if isControlled(vObj) || f.isExcluded(pObj) {
		return ctrl.Result{}, nil
	} else if !f.objectMatches(vObj) {
		ctx.Log.Infof("delete physical %s %s/%s, because it is not used anymore", f.config.Kind, pObj.GetNamespace(), pObj.GetName())
		err := ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil {
			ctx.Log.Infof("error deleting physical %s %s/%s in physical cluster: %v", f.config.Kind, pObj.GetNamespace(), pObj.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// apply reverse patches
	klog.Infof("applying reverse patches")
	result, err := f.patcher.ApplyReversePatches(ctx.Context, vObj, pObj, f.config.ReversePatches, &hostToVirtualNameResolver{
		gvk:  f.gvk,
		pObj: pObj,
	})
	if err != nil {
		if kerrors.IsInvalid(err) {
			ctx.Log.Infof("Warning: this message could indicate a timing issue with no significant impact, or a bug. Please report this if your resource never reaches the expected state. Error message: failed to patch virtual %s %s/%s: %v", f.config.Kind, vObj.GetNamespace(), vObj.GetName(), err)
			// this happens when some field is being removed shortly after being added, which suggest it's a timing issue
			// it doesn't seem to have any negative consequence besides the logged error message
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to virtual cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("failed to patch virtual %s %s/%s: %v", f.config.Kind, vObj.GetNamespace(), vObj.GetName(), err)
	} else if result == controllerutil.OperationResultUpdated || result == controllerutil.OperationResultUpdatedStatus || result == controllerutil.OperationResultUpdatedStatusOnly {
		// a change will trigger reconciliation anyway, and at that point we can make
		// a more accurate updates(reverse patches) to the virtual resource
		return ctrl.Result{}, nil
	}

	// apply patches
	_, err = f.patcher.ApplyPatches(ctx.Context, vObj, pObj, f.config.Patches, f.config.ReversePatches, func(vObj client.Object) (client.Object, error) {
		return f.TranslateMetadata(vObj), nil
	}, &virtualToHostNameResolver{
		namespace:       vObj.GetNamespace(),
		targetNamespace: translate.Default.PhysicalNamespace(vObj.GetNamespace())})
	if err != nil {
		if kerrors.IsInvalid(err) {
			ctx.Log.Infof("Warning: this message could indicate a timing issue with no significant impact, or a bug. Please report this if your resource never reaches the expected state. Error message: failed to patch physical %s %s/%s: %v", f.config.Kind, vObj.GetNamespace(), vObj.GetName(), err)
			// this happens when some field is being removed shortly after being added, which suggest it's a timing issue
			// it doesn't seem to have any negative consequence besides the logged error message
			return ctrl.Result{Requeue: true}, nil
		}

		f.EventRecorder().Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, fmt.Errorf("error applying patches: %v", err)
	}

	return ctrl.Result{}, nil
}

var _ syncer.UpSyncer = &exporter{}

func (f *exporter) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	if !translate.Default.IsManaged(pObj) || f.isExcluded(pObj) {
		return ctrl.Result{}, nil
	}

	// delete physical object because virtual one is missing
	return syncer.DeleteObject(ctx, pObj, fmt.Sprintf("delete physical %s because virtual is missing", pObj.GetName()))
}

func (f *exporter) getControllerID() string {
	if f.config.ID != "" {
		return f.config.ID
	}

	return strings.Join(append(strings.Split(f.config.APIVersion, "/"), f.config.Kind), "-")
}

func (f *exporter) Name() string {
	return f.getControllerID()
}

// TranslateMetadata converts the virtual object into a physical object
func (f *exporter) TranslateMetadata(vObj client.Object) client.Object {
	pObj := f.NamespacedTranslator.TranslateMetadata(vObj)
	labels := pObj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	labels[translate.ControllerLabel] = f.getControllerID()
	pObj.SetLabels(labels)
	return pObj
}

func (f *exporter) IsManaged(pObj client.Object) (bool, error) {
	if !translate.Default.IsManaged(pObj) {
		return false, nil
	}

	return !f.isExcluded(pObj), nil
}

func isControlled(obj client.Object) bool {
	return obj.GetLabels() != nil && obj.GetLabels()[translate.ControllerLabel] != ""
}

func (f *exporter) objectMatches(obj client.Object) bool {
	return f.selector == nil || !f.selector.Matches(labels.Set(obj.GetLabels()))
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
	} else {
		return translate.Default.PhysicalName(name, namespace), nil
	}
}

func (r *virtualToHostNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return translate.Default.TranslateLabelSelectorCluster(selector), nil
}

func (r *virtualToHostNameResolver) TranslateLabelKey(key string) (string, error) {
	return translate.Default.ConvertLabelKey(key), nil
	// return key, nil
}

func (r *virtualToHostNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	s := map[string]string{}
	if selector != nil {
		for k, v := range selector {
			s[k] = v
		}
		s[translate.NamespaceLabel] = r.namespace
		s[translate.MarkerLabel] = translate.Suffix
	}
	return s, nil
}

func (r *virtualToHostNameResolver) TranslateNamespaceRef(namespace string) (string, error) {
	return translate.Default.PhysicalNamespace(namespace), nil
}

func validateExportConfig(config *config.Export) error {
	for _, p := range append(config.Patches, config.ReversePatches...) {
		if p.Regex != "" {
			parsed, err := patchesregex.PrepareRegex(p.Regex)
			if err != nil {
				return fmt.Errorf("invalid Regex: %v", err)
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

func (r *hostToVirtualNameResolver) TranslateName(name string, regex *regexp.Regexp, path string) (string, error) {
	var n types.NamespacedName
	// if regex != nil {
	// 	return patchesregex.ProcessRegex(regex, name, func(name, namespace string) types.NamespacedName {
	// 		if path == "" {
	// 			return r.nameCache.ResolveName(r.gvk, name)
	// 		} else {
	// 			return r.nameCache.ResolveNamePath(r.gvk, name, path)
	// 		}
	// 	}), nil
	// } else {
	// 	if path == "" {
	// 		n = r.nameCache.ResolveName(r.gvk, name)
	// 	} else {
	// 		n = r.nameCache.ResolveNamePath(r.gvk, name, path)
	// 	}
	// }
	if n.Name == "" {
		return "", fmt.Errorf("could not translate %s host resource name to vcluster resource name", name)
	}

	klog.Info("************************* translated name ********************")
	annotations := r.pObj.GetAnnotations()
	if name, ok := annotations[VclusterTranslationObjectNameKey]; ok {
		klog.Info("name: ", name)
	} else {
		klog.Error("cannot find key", VclusterTranslationObjectNameKey)
	}

	return n.Name, nil
}
func (r *hostToVirtualNameResolver) TranslateNameWithNamespace(name string, namespace string, regex *regexp.Regexp, path string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelKey(key string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelExpressionsSelector(selector *metav1.LabelSelector) (*metav1.LabelSelector, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateLabelSelector(selector map[string]string) (map[string]string, error) {
	return nil, fmt.Errorf("translation not supported from host to virtual object")
}
func (r *hostToVirtualNameResolver) TranslateNamespaceRef(namespace string) (string, error) {
	return "", fmt.Errorf("translation not supported from host to virtual object")
}

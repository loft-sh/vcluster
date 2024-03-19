package translator

import (
	context2 "context"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterTranslator(ctx *context.RegisterContext, name string, obj client.Object, nameTranslator translate.PhysicalNameTranslator, excludedAnnotations ...string) Translator {
	return &clusterTranslator{
		name:                name,
		excludedAnnotations: excludedAnnotations,
		virtualClient:       ctx.VirtualManager.GetClient(),
		obj:                 obj,
		nameTranslator:      nameTranslator,
		syncedLabels:        ctx.Config.Experimental.SyncSettings.SyncLabels,
	}
}

type clusterTranslator struct {
	name                string
	virtualClient       client.Client
	obj                 client.Object
	nameTranslator      translate.PhysicalNameTranslator
	excludedAnnotations []string
	syncedLabels        []string
}

func (n *clusterTranslator) Name() string {
	return n.name
}

func (n *clusterTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *clusterTranslator) IsManaged(_ context2.Context, pObj client.Object) (bool, error) {
	return translate.Default.IsManagedCluster(pObj), nil
}

func (n *clusterTranslator) VirtualToHost(_ context2.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name: n.nameTranslator(req.Name, vObj),
	}
}

func (n *clusterTranslator) HostToVirtual(ctx context2.Context, req types.NamespacedName, pObj client.Object) types.NamespacedName {
	if pObj != nil {
		pAnnotations := pObj.GetAnnotations()
		if pAnnotations != nil && pAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: pAnnotations[translate.NamespaceAnnotation],
				Name:      pAnnotations[translate.NameAnnotation],
			}
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(ctx, n.virtualClient, vObj, constants.IndexByPhysicalName, req.Name)
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}

func (n *clusterTranslator) TranslateMetadata(_ context2.Context, vObj client.Object) client.Object {
	pObj, err := translate.Default.SetupMetadataWithName(vObj, n.nameTranslator)
	if err != nil {
		return nil
	}

	pObj.SetLabels(n.TranslateLabels(vObj, nil))
	pObj.SetAnnotations(n.TranslateAnnotations(vObj, nil))
	return pObj
}

func (n *clusterTranslator) TranslateMetadataUpdate(_ context2.Context, vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := n.TranslateAnnotations(vObj, pObj)
	updatedLabels := n.TranslateLabels(vObj, pObj)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (n *clusterTranslator) TranslateLabels(vObj client.Object, pObj client.Object) map[string]string {
	return translate.Default.TranslateLabelsCluster(vObj, pObj, n.syncedLabels)
}

func (n *clusterTranslator) TranslateAnnotations(vObj client.Object, pObj client.Object) map[string]string {
	return translate.Default.ApplyAnnotations(vObj, pObj, n.excludedAnnotations)
}

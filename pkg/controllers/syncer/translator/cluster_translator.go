package translator

import (
	context2 "context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterTranslator(ctx *context.RegisterContext, name string, obj client.Object, mapper mappings.Mapper, excludedAnnotations ...string) Translator {
	return &clusterTranslator{
		Mapper: mapper,

		name: name,

		excludedAnnotations: excludedAnnotations,
		virtualClient:       ctx.VirtualManager.GetClient(),
		obj:                 obj,
		syncedLabels:        ctx.Config.Experimental.SyncSettings.SyncLabels,
	}
}

type clusterTranslator struct {
	mappings.Mapper

	name string

	virtualClient       client.Client
	obj                 client.Object
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

func (n *clusterTranslator) TranslateMetadata(ctx context2.Context, vObj client.Object) client.Object {
	pObj, err := translate.Default.SetupMetadataWithName(vObj, n.Mapper.VirtualToHost(ctx, types.NamespacedName{Name: vObj.GetName(), Namespace: vObj.GetNamespace()}, vObj))
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

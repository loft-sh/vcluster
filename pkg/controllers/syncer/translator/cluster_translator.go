package translator

import (
	context2 "context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClusterTranslator(ctx *context.RegisterContext, name string, obj client.Object, nameTranslator PhysicalNameTranslator, excludedAnnotations ...string) Translator {
	return &clusterTranslator{
		name:                name,
		physicalNamespace:   ctx.TargetNamespace,
		excludedAnnotations: excludedAnnotations,
		virtualClient:       ctx.VirtualManager.GetClient(),
		obj:                 obj,
		nameTranslator:      nameTranslator,
		syncedLabels:        ctx.Options.SyncLabels,
	}
}

type clusterTranslator struct {
	name                string
	physicalNamespace   string
	virtualClient       client.Client
	obj                 client.Object
	nameTranslator      PhysicalNameTranslator
	excludedAnnotations []string
	syncedLabels        []string
}

func (n *clusterTranslator) Name() string {
	return n.name
}

func (n *clusterTranslator) Resource() client.Object {
	return n.obj.DeepCopyObject().(client.Object)
}

func (n *clusterTranslator) IsManaged(pObj client.Object) (bool, error) {
	return translate.IsManagedCluster(n.physicalNamespace, pObj), nil
}

func (n *clusterTranslator) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Name: n.nameTranslator(req.Name, vObj),
	}
}

func (n *clusterTranslator) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	pAnnotations := pObj.GetAnnotations()
	if pAnnotations != nil && pAnnotations[NameAnnotation] != "" {
		return types.NamespacedName{
			Namespace: pAnnotations[NamespaceAnnotation],
			Name:      pAnnotations[NameAnnotation],
		}
	}

	vObj := n.obj.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(context2.Background(), n.virtualClient, vObj, constants.IndexByPhysicalName, pObj.GetName())
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: vObj.GetNamespace(),
		Name:      vObj.GetName(),
	}
}

func (n *clusterTranslator) TranslateMetadata(vObj client.Object) client.Object {
	pObj, err := setupMetadataWithName(n.physicalNamespace, vObj, n.nameTranslator)
	if err != nil {
		return nil
	}

	pObj.SetLabels(n.TranslateLabels(vObj, nil))
	pObj.SetAnnotations(n.TranslateAnnotations(vObj, nil))
	return pObj
}

func (n *clusterTranslator) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := n.TranslateAnnotations(vObj, pObj)
	updatedLabels := n.TranslateLabels(vObj, pObj)
	return !equality.Semantic.DeepEqual(updatedAnnotations, pObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, pObj.GetLabels()), updatedAnnotations, updatedLabels
}

func (n *clusterTranslator) TranslateLabels(vObj client.Object, pObj client.Object) map[string]string {
	newLabels := map[string]string{}
	if vObj != nil {
		vObjLabels := vObj.GetLabels()
		for k, v := range vObjLabels {
			newLabels[convertNamespacedLabelKey(n.physicalNamespace, k)] = v
		}
		if vObjLabels != nil {
			for _, k := range n.syncedLabels {
				if value, ok := vObjLabels[k]; ok {
					newLabels[k] = value
				}
			}
		}
	}
	if pObj != nil {
		pObjLabels := pObj.GetLabels()
		if pObjLabels != nil && pObjLabels[translate.ControllerLabel] != "" {
			newLabels[translate.ControllerLabel] = pObjLabels[translate.ControllerLabel]
		}
	}
	newLabels[translate.MarkerLabel] = translate.SafeConcatName(n.physicalNamespace, "x", translate.Suffix)
	return newLabels
}

func (n *clusterTranslator) TranslateAnnotations(vObj client.Object, pObj client.Object) map[string]string {
	return translateAnnotations(vObj, pObj, n.excludedAnnotations)
}

func convertNamespacedLabelKey(physicalNamespace, key string) string {
	digest := sha256.Sum256([]byte(key))
	return translate.SafeConcatName(LabelPrefix, physicalNamespace, "x", translate.Suffix, "x", hex.EncodeToString(digest[0:])[0:10])
}

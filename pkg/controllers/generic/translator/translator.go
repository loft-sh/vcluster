package translator

import (
	"github.com/loft-sh/vcluster/pkg/controllers/generic/context"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Translator is used to translate names as well as metadata between virtual and physical objects
type Translator interface {
	NameTranslator
	MetadataTranslator
}

// NameTranslator is used to convert virtual to physical names and vice versa
type NameTranslator interface {
	// IsManaged determines if a physical object is managed by the vcluster
	IsManaged(pObj client.Object) (bool, error)

	// VirtualToPhysical translates a virtual name to a physical name
	VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName

	// PhysicalToVirtual translates a physical name to a virtual name
	PhysicalToVirtual(pObj client.Object) types.NamespacedName
}

// MetadataTranslator is used to convert metadata between virtual and physical objects and vice versa
type MetadataTranslator interface {
	// TranslateMetadata translates the object's metadata
	TranslateMetadata(vObj client.Object) client.Object

	// TranslateMetadataUpdate translates the object's metadata annotations and labels and determines
	// if they have changed between the physical and virtual object
	TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string)
}

// ForwardTranslator provides some helper functions to ease forward translation
type ForwardTranslator interface {
	Translator

	// ForwardCreate creates the given pObj in the target namespace
	ForwardCreate(ctx context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error)

	// ForwardUpdate updates the given pObj (if not nil) in the target namespace
	ForwardUpdate(ctx context.SyncContext, vObj, pObj client.Object) (ctrl.Result, error)
}

// PhysicalNameTranslator transforms a virtual cluster name to a physical name
type PhysicalNameTranslator func(vName string, vObj client.Object) string

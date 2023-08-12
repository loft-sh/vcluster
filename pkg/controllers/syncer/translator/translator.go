package translator

import (
	"context"

	syncercontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Translator is used to translate names as well as metadata between virtual and physical objects
type Translator interface {
	Resource() client.Object
	Name() string
	NameTranslator
	MetadataTranslator
}

// NameTranslator is used to convert virtual to physical names and vice versa
type NameTranslator interface {
	// IsManaged determines if a physical object is managed by the vcluster
	IsManaged(context.Context, client.Object) (bool, error)

	// VirtualToPhysical translates a virtual name to a physical name
	VirtualToPhysical(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName

	// PhysicalToVirtual translates a physical name to a virtual name
	PhysicalToVirtual(ctx context.Context, pObj client.Object) types.NamespacedName
}

// MetadataTranslator is used to convert metadata between virtual and physical objects and vice versa
type MetadataTranslator interface {
	// TranslateMetadata translates the object's metadata
	TranslateMetadata(ctx context.Context, vObj client.Object) client.Object

	// TranslateMetadataUpdate translates the object's metadata annotations and labels and determines
	// if they have changed between the physical and virtual object
	TranslateMetadataUpdate(ctx context.Context, vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string)
}

// NamespacedTranslator provides some helper functions to ease sync down translation
type NamespacedTranslator interface {
	Translator

	// EventRecorder returns
	EventRecorder() record.EventRecorder

	// RegisterIndices registers the default indices for the syncer
	RegisterIndices(ctx *syncercontext.RegisterContext) error

	// SyncDownCreate creates the given pObj in the target namespace
	SyncDownCreate(ctx *syncercontext.SyncContext, vObj, pObj client.Object) (ctrl.Result, error)

	// SyncDownUpdate updates the given pObj (if not nil) in the target namespace
	SyncDownUpdate(ctx *syncercontext.SyncContext, vObj, pObj client.Object) (ctrl.Result, error)

	// Function to override default VirtualToPhysical name translation
	SetNameTranslator(nameTranslator translate.PhysicalNamespacedNameTranslator)
}

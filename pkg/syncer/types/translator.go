package types

import (
	syncercontext "github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/client-go/tools/record"
)

// Translator is used to translate names as well as metadata between virtual and physical objects
type Translator interface {
	Object
	syncercontext.Mapper
}

// GenericTranslator provides some helper functions to ease sync down translation
type GenericTranslator interface {
	Translator

	// EventRecorder returns
	EventRecorder() record.EventRecorder
}

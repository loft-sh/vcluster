package pro

import (
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

var GetNamespaceMapper = func(_ *synccontext.RegisterContext, _ synccontext.Mapper) (synccontext.Mapper, error) {
	return nil, NewFeatureError("namespace toHost syncing")
}

var GetWithSyncedNamespacesTranslator = func(_ string, _ config.FromHostMappings) (translate.Translator, error) {
	return nil, NewFeatureError("namespace toHost syncing")
}

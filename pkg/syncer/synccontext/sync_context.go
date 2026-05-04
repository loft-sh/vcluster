package synccontext

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SyncContext struct {
	context.Context

	Log loghelper.Logger

	Config *config.VirtualClusterConfig

	HostClient    client.Client
	VirtualClient client.Client

	ObjectCache *BidirectionalObjectCache

	Mappings MappingsRegistry

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
}

func (s *SyncContext) Close() error {
	if s.Mappings != nil && s.Mappings.Store() != nil {
		// check if we have the owning object in the context
		belongsTo, ok := MappingFrom(s.Context)
		if !ok {
			return nil
		}

		// save the mapping in the store
		err := s.Mappings.Store().SaveMapping(s, belongsTo)
		if err != nil {
			return fmt.Errorf("save mapping: %w", err)
		}
	}

	return nil
}

type syncContextMappingType int

const mappingKey syncContextMappingType = iota

// WithMappingFromObjects adds the mapping to the context
func WithMappingFromObjects(ctx context.Context, pObj, vObj client.Object) (context.Context, error) {
	nameMapping, err := NewNameMappingFrom(pObj, vObj)
	if err != nil {
		return nil, err
	}

	return WithMapping(ctx, nameMapping), nil
}

// WithMapping adds the mapping to the context
func WithMapping(ctx context.Context, nameMapping NameMapping) context.Context {
	if nameMapping.Empty() {
		return ctx
	}

	return context.WithValue(ctx, mappingKey, nameMapping)
}

// WithoutMapping returns a copy of ctx that preserves cancellation and deadlines but hides the current mapping.
// Use this when code needs to translate a referenced object as a probe before deciding whether to record that dependency.
// For example, route syncers first resolve a referenced Gateway or Service without recorder side effects, validate the host object,
// and then record the dependency intentionally.
func WithoutMapping(ctx *SyncContext) *SyncContext {
	if ctx == nil {
		return nil
	}

	noMappingCtx := *ctx
	baseCtx := ctx.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	noMappingCtx.Context = context.WithValue(baseCtx, mappingKey, nil)
	return &noMappingCtx
}

// MappingFrom returns the value of the original request path key on the ctx
func MappingFrom(ctx context.Context) (NameMapping, bool) {
	info, ok := ctx.Value(mappingKey).(NameMapping)
	if info.Empty() {
		return NameMapping{}, false
	}
	return info, ok
}

package generic

import "k8s.io/apimachinery/pkg/runtime/schema"

type GVKRegister map[schema.GroupVersionKind]*GVKScopeAndSubresource

type GVKScopeAndSubresource struct {
	IsClusterScoped      bool
	HasStatusSubresource bool
}

package generic

import "k8s.io/apimachinery/pkg/runtime/schema"

type ClusterScopedGVKRegister map[schema.GroupVersionKind]bool

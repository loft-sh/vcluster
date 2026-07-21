/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	"sigs.k8s.io/e2e-framework/pkg/types"
)

type (
	ClusterOpts                       = types.ClusterOpts
	Node                              = types.Node
	NodeOperation                     = types.NodeOperation
	ClusterNameContextKey             = types.ClusterNameContextKey
	E2EClusterProvider                = types.E2EClusterProvider
	E2EClusterProviderWithImageLoader = types.E2EClusterProviderWithImageLoader
	E2EClusterProviderWithLifeCycle   = types.E2EClusterProviderWithLifeCycle
)

const (
	AddNode    = types.AddNode
	RemoveNode = types.RemoveNode
	StartNode  = types.StartNode
	StopNode   = types.StopNode
)

/*
Copyright 2024 The Kubernetes Authors.

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

package featuregate

import (
	"k8s.io/component-base/featuregate"
)

var (
	FeatureGate = featuregate.NewFeatureGate()

	DefaultMutableFeatureGate featuregate.MutableFeatureGate = FeatureGate

	DefaultFeatureGate featuregate.FeatureGate = DefaultMutableFeatureGate
)

const (
	ReverseTestFinishExecutionOrder = featuregate.Feature("ReverseTestFinishExecutionOrder")
)

var defaultE2EFrakeworkFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	ReverseTestFinishExecutionOrder: {Default: false, PreRelease: featuregate.Alpha},
}

func init() {
	if err := DefaultMutableFeatureGate.Add(defaultE2EFrakeworkFeatureGates); err != nil {
		panic(err)
	}
}

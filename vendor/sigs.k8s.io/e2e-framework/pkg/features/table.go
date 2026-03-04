/*
Copyright 2021 The Kubernetes Authors.

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

package features

import (
	"fmt"
)

type TableRow struct {
	Name        string
	Description string
	Assessment  Func
}

// Table provides a structure for table-driven tests.
// Each entry in the table represents an executable assessment.
type Table []TableRow

// Build converts the defined test steps in the table
// into a FeatureBuilder which can be used to add additional attributes
// to the feature before it's exercised. Build takes an optional feature name
// if omitted will be generated.
func (table Table) Build(args ...string) *FeatureBuilder {
	var name string
	var description string
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		description = args[1]
	}
	f := NewWithDescription(name, description)
	for i, test := range table {
		if test.Name == "" {
			test.Name = fmt.Sprintf("Assessment-%d", i)
		}
		if test.Assessment != nil {
			f.AssessWithDescription(test.Name, test.Description, test.Assessment)
		}
	}
	return f
}

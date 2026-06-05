/*
Copyright 2022 The Kubernetes Authors.

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

package validation_test

import (
	"testing"

	gatewayv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	validationutil "sigs.k8s.io/gateway-api/apis/v1beta1/util/validation"
)

func TestIsControllerNameValid(t *testing.T) {
	testCases := []struct {
		name           string
		controllerName gatewayv1b1.GatewayController
		isvalid        bool
	}{
		{
			name:           "empty controller name",
			controllerName: "",
			isvalid:        false,
		},
		{
			name:           "invalid controller name 1",
			controllerName: "example.com",
			isvalid:        false,
		},
		{
			name:           "invalid controller name 2",
			controllerName: "example*com/bar",
			isvalid:        false,
		},
		{
			name:           "invalid controller name 3",
			controllerName: "example/@bar",
			isvalid:        false,
		},
		{
			name:           "valid controller name",
			controllerName: "example.com/bar",
			isvalid:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := validationutil.IsControllerNameValid(tc.controllerName)
			if isValid != tc.isvalid {
				t.Errorf("Expected validity %t, got %t", tc.isvalid, isValid)
			}
		})
	}
}

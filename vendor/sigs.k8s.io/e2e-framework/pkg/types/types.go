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

package types

import (
	"context"
	"net"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/flags"
)

// EnvFunc represents a user-defined operation that
// can be used to customize the behavior of the
// environment. Changes to context are expected to surface
// to caller.
type EnvFunc func(context.Context, *envconf.Config) (context.Context, error)

// FeatureEnvFunc represents a user-defined operation that
// can be used to customize the behavior of the
// environment. Changes to context are expected to surface
// to caller. Meant for use with before/after feature hooks.
// *testing.T is provided in order to provide pass/fail context to
// features.
type FeatureEnvFunc func(context.Context, *envconf.Config, *testing.T, Feature) (context.Context, error)

// TestEnvFunc represents a user-defined operation that
// can be used to customize the behavior of the
// environment. Changes to context are expected to surface
// to caller. Meant for use with before/after test hooks.
type TestEnvFunc func(context.Context, *envconf.Config, *testing.T) (context.Context, error)

// Environment represents an environment where
// features can be tested.
type Environment interface {
	// WithContext returns a new Environment with a new context
	WithContext(context.Context) Environment

	// Setup registers environment operations that are executed once
	// prior to the environment being ready and prior to any test.
	Setup(...EnvFunc) Environment

	// BeforeEachTest registers environment funcs that are executed
	// before each Env.Test(...)
	BeforeEachTest(...TestEnvFunc) Environment

	// BeforeEachFeature registers step functions that are executed
	// before each Feature is tested during env.Test call.
	BeforeEachFeature(...FeatureEnvFunc) Environment

	// AfterEachFeature registers step functions that are executed
	// after each feature is tested during an env.Test call.
	AfterEachFeature(...FeatureEnvFunc) Environment

	// Test executes a test feature defined in a TestXXX function
	// This method surfaces context for further updates.
	Test(*testing.T, ...Feature) context.Context

	// TestInParallel executes a series of test features defined in a
	// TestXXX function in parallel. This works the same way Test method
	// does with the caveat that the features will all be run in parallel
	TestInParallel(*testing.T, ...Feature) context.Context

	// AfterEachTest registers environment funcs that are executed
	// after each Env.Test(...).
	AfterEachTest(...TestEnvFunc) Environment

	// Finish registers funcs that are executed at the end of the
	// test suite.
	Finish(...EnvFunc) Environment

	// Run Launches the test suite from within a TestMain
	Run(*testing.M) int

	// EnvConf returns the test environment's environment configuration
	EnvConf() *envconf.Config
}

type Labels = flags.LabelsMap

type Feature interface {
	// Name is a descriptive text for the feature
	Name() string
	// Labels returns a map of feature labels
	Labels() Labels
	// Steps testing tasks to test the feature
	Steps() []Step
}

type Level uint8

const (
	// LevelSetup when doing the setup phase
	LevelSetup Level = iota
	// LevelAssess when doing the assess phase
	LevelAssess
	// LevelTeardown when doing the teardown phase
	LevelTeardown
)

type StepFunc func(context.Context, *testing.T, *envconf.Config) context.Context

type Step interface {
	// Name is the step name
	Name() string
	// Level action level {setup|requirement|assertion|teardown}
	Level() Level
	// Func is the operation for the step
	Func() StepFunc
}

type DescribableStep interface {
	Step
	// Description is the Readable test description indicating the purpose behind the test that
	// can add more context to the test under question
	Description() string
}

type DescribableFeature interface {
	Feature

	// Description is used to provide a readable context for the test feature. This can be used
	// to provide more context for the test being performed and the assessment under each of the
	// feature.
	Description() string
}

type ClusterOpts func(c E2EClusterProvider)

type Node struct {
	Name    string
	Role    string
	Cluster string
	State   string
	IP      net.IP
}

type NodeOperation string

const (
	AddNode    NodeOperation = "add"
	RemoveNode NodeOperation = "remove"
	StartNode  NodeOperation = "start"
	StopNode   NodeOperation = "stop"
)

type ClusterNameContextKey string

type E2EClusterProvider interface {
	// WithName is used to configure the cluster Name that should be used while setting up the cluster. Might
	// Not apply for all providers.
	WithName(name string) E2EClusterProvider

	// WithVersion helps you override the default version used while using the cluster provider.
	// This can be useful in providing a mechanism to the end users where they want to test their
	// code against a certain specific version of k8s that is not the default one configured
	// for the provider
	WithVersion(version string) E2EClusterProvider

	// WithPath heps you customize the executable binary that is used to back the cluster provider.
	// This is useful in cases where your binary is present in a non standard location output of the
	// PATH variable and you want to use that instead of framework trying to install one on it's own.
	WithPath(path string) E2EClusterProvider

	// WithOpts provides a way to customize the options that can be used while setting up the
	// cluster using the providers such as kind or kwok or anything else. These helpers can be
	// leveraged to setup arguments or configuration values that can be provided while performing
	// the cluster bring up
	WithOpts(opts ...ClusterOpts) E2EClusterProvider

	// Create Provides an interface to start the cluster creation workflow using the selected provider
	Create(ctx context.Context, args ...string) (string, error)

	// CreateWithConfig is used to provide a mechanism where cluster providers that take an input config
	// file and then setup the cluster accordingly. This can be used to provide input such as kind config
	CreateWithConfig(ctx context.Context, configFile string) (string, error)

	// GetKubeconfig provides a way to extract the kubeconfig file associated with the cluster in question
	// using the cluster provider native way
	GetKubeconfig() string

	// GetKubectlContext is used to extract the kubectl context to be used while performing the operation
	GetKubectlContext() string

	// ExportLogs is used to export the cluster logs via the cluster provider native workflow. This
	// can be used to export logs from the cluster after test failures for example to analyze the test
	// failures better after the fact.
	ExportLogs(ctx context.Context, dest string) error

	// Destroy is used to cleanup a cluster brought up as part of the test workflow
	Destroy(ctx context.Context) error

	// SetDefaults is a handler function invoked after creating an object of type E2EClusterProvider. This method is
	// invoked as the first step after creating an object in order to make sure the default values for required
	// attributes are setup accordingly if any.
	SetDefaults() E2EClusterProvider

	// WaitForControlPlane is a helper function that can be used to indiate the Provider based cluster create workflow
	// that the control plane is fully up and running. This method is invoked after the Create/CreateWithConfig handlers
	// and is expected to return an error if the control plane doesn't stabilize. If the provider being implemented
	// does not have a clear mechanism to identify the Control plane readiness or is not required to wait for the control
	// plane to be ready, such providers can simply add a no-op workflow for this function call.
	// Returning an error message from this handler will stop the workflow of e2e-framework as returning an error from this
	// is considered as  failure to provision a cluster
	WaitForControlPlane(ctx context.Context, client klient.Client) error

	// KubernetesRestConfig is a helper function that provides an instance of rest.Config which can then be used to
	// create your own clients if you chose to do so.
	KubernetesRestConfig() *rest.Config
}

type E2EClusterProviderWithImageLoader interface {
	E2EClusterProvider

	// LoadImage is used to load a set of Docker images to the cluster via the cluster provider native workflow
	// Not every provider will have a mechanism like this/need to do this. So, providers that do not have this support
	// can just provide a no-op implementation to be compliant with the interface
	LoadImage(ctx context.Context, image string, args ...string) error

	// LoadImageArchive is used to provide a mechanism where a tar.gz archive containing the docker images used
	// by the services running on the cluster can be imported and loaded into the cluster prior to the execution of
	// test if required.
	// Not every provider will have a mechanism like this/need to do this. So, providers that do not have this support
	// can just provide a no-op implementation to be compliant with the interface
	LoadImageArchive(ctx context.Context, archivePath string, args ...string) error
}

// E2EClusterProviderWithLifeCycle is an interface that extends the E2EClusterProviderWithImageLoader
// interface to provide a mechanism to add/remove nodes from the cluster as part of the E2E Test workflow.
//
// This can be useful while performing the e2e test that revolves around the node lifecycle events.
// eg: You have a kubernetes controller that acts upon the v1.Node resource of the k8s and you want to
// test out how the Remove operation impacts your workflow.
// Or you want to simulate a case where one or more node of your cluster is down and you want to see how
// your application reacts to such failure events.
type E2EClusterProviderWithLifeCycle interface {
	E2EClusterProvider

	// AddNode is used to add a new node to the existing cluster as part of the E2E Test workflow.
	// Not every provider will have a mechanism to support this. e.g Kind. But k3d has support for this.
	// This will be implemented as an optional interface depending on the provider in question.
	AddNode(ctx context.Context, node *Node, args ...string) error

	// RemoveNode can be used to remove a node from an existing cluster as part of the E2E Test workflow.
	// Not every provider will have a mechanism to support this. e.g Kind. But k3d has support for this.
	// This will be implemented as an optional interface depending on the provider in question.
	RemoveNode(ctx context.Context, node *Node, args ...string) error

	// StartNode is used to start a node that was shutdown/powered down as part of the E2E Test workflow.
	// Not every provider will have a mechanism to support this. e.g Kind. But k3d has support for this.
	// This will be implemented as an optional interface depending on the provider in question.
	StartNode(ctx context.Context, node *Node, args ...string) error

	// StopNode can be used to stop an running node from the cluster as part of the E2E test Workflow.
	// Not every provider will have a mechanism to support this. e.g Kind. But k3d has support for this.
	// This will be implemented as an optional interface depending on the provider in question.
	StopNode(ctx context.Context, node *Node, args ...string) error

	// ListNode can be used to fetch the list of nodes in the cluster. This can be used to extract the
	// List of existing nodes on the cluster and their state before they can be operated on.
	ListNode(ctx context.Context, args ...string) ([]Node, error)
}

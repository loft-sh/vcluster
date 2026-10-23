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

// Package env exposes types to create type `Environment` used to run
// feature tests.
package env

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"
	"sort"
	"sync"
	"testing"

	klog "k8s.io/klog/v2"

	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/featuregate"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

type (
	Environment = types.Environment
	Func        = types.EnvFunc
	FeatureFunc = types.FeatureEnvFunc
	TestFunc    = types.TestEnvFunc
)

type testEnv struct {
	ctx     context.Context
	cfg     *envconf.Config
	actions []action
}

// New creates a test environment with no config attached.
func New() types.Environment {
	return newTestEnv()
}

func NewParallel() types.Environment {
	return newTestEnvWithParallel()
}

// NewWithConfig creates an environment using an Environment Configuration value
func NewWithConfig(cfg *envconf.Config) types.Environment {
	env := newTestEnv()
	env.cfg = cfg
	return env
}

// NewFromFlags creates a test environment using configuration values from CLI flags
func NewFromFlags() (types.Environment, error) {
	cfg, err := envconf.NewFromFlags()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg), nil
}

// NewWithKubeConfig creates an environment using an Environment Configuration value
// and the given kubeconfig.
func NewWithKubeConfig(kubeconfigfile string) types.Environment {
	env := newTestEnv()
	cfg := envconf.NewWithKubeConfig(kubeconfigfile)
	env.cfg = cfg
	return env
}

// NewInClusterConfig creates an environment using an Environment Configuration value
// and assumes an in-cluster kubeconfig.
func NewInClusterConfig() types.Environment {
	env := newTestEnv()
	cfg := envconf.NewWithKubeConfig("")
	env.cfg = cfg
	return env
}

// NewWithContext creates a new environment with the provided context and config.
func NewWithContext(ctx context.Context, cfg *envconf.Config) (types.Environment, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("environment config is nil")
	}
	return &testEnv{ctx: ctx, cfg: cfg}, nil
}

func newTestEnv() *testEnv {
	return &testEnv{
		ctx: context.Background(),
		cfg: envconf.New(),
	}
}

func newTestEnvWithParallel() *testEnv {
	return &testEnv{
		ctx: context.Background(),
		cfg: envconf.New().WithParallelTestEnabled(),
	}
}

type ctxName string

// newChildTestEnv returns a child testEnv based on the one passed as an argument.
// The child env inherits the context and actions from the parent and
// creates a deep copy of the config so that it can be mutated without
// affecting the parent's.
func newChildTestEnv(e *testEnv) *testEnv {
	childCtx := context.WithValue(e.ctx, ctxName("parent"), fmt.Sprintf("%s", e.ctx))
	return &testEnv{
		ctx:     childCtx,
		cfg:     e.deepCopyConfig(),
		actions: append([]action{}, e.actions...),
	}
}

// WithContext returns a new environment with the context set to ctx.
// Argument ctx cannot be nil
func (e *testEnv) WithContext(ctx context.Context) types.Environment {
	if ctx == nil {
		panic("nil context") // this should never happen
	}
	env := &testEnv{
		ctx: ctx,
		cfg: e.cfg,
	}
	env.actions = append(env.actions, e.actions...)
	return env
}

// Setup registers environment operations that are executed once
// prior to the environment being ready and prior to any test.
func (e *testEnv) Setup(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleSetup, funcs: funcs})
	return e
}

// BeforeEachTest registers environment funcs that are executed
// before each Env.Test(...)
func (e *testEnv) BeforeEachTest(funcs ...types.TestEnvFunc) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleBeforeTest, testFuncs: funcs})
	return e
}

// BeforeEachFeature registers step functions that are executed
// before each Feature is tested during env.Test call.
func (e *testEnv) BeforeEachFeature(funcs ...FeatureFunc) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleBeforeFeature, featureFuncs: funcs})
	return e
}

// AfterEachFeature registers step functions that are executed
// after each feature is tested during an env.Test call.
func (e *testEnv) AfterEachFeature(funcs ...FeatureFunc) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleAfterFeature, featureFuncs: funcs})
	return e
}

// AfterEachTest registers environment funcs that are executed
// after each Env.Test(...).
func (e *testEnv) AfterEachTest(funcs ...types.TestEnvFunc) types.Environment {
	if len(funcs) == 0 {
		return e
	}
	e.actions = append(e.actions, action{role: roleAfterTest, testFuncs: funcs})
	return e
}

// panicOnMissingContext is used to check if the test Env has a non-nil context setup
// and fail fast if the context has not already been set
func (e *testEnv) panicOnMissingContext() {
	if e.ctx == nil {
		panic("context not set") // something is terribly wrong.
	}
}

// processTestActions is used to run a series of test action that were configured as
// BeforeEachTest or AfterEachTest
func (e *testEnv) processTestActions(ctx context.Context, t *testing.T, actions []action) context.Context {
	t.Helper()
	var err error
	out := ctx
	for _, action := range actions {
		out, err = action.runWithT(out, e.cfg, t)
		if err != nil {
			t.Fatalf("%s failure: %s", action.role, err)
		}
	}
	return out
}

// processTestFeature is used to trigger the execution of the actual feature. This function wraps the entire
// workflow of orchestrating the feature execution be running the action configured by BeforeEachFeature /
// AfterEachFeature.
func (e *testEnv) processTestFeature(ctx context.Context, t *testing.T, featureName string, feature types.Feature) context.Context {
	t.Helper()
	skipped, message := e.requireFeatureProcessing(feature)
	if skipped {
		t.Skip(message)
	}
	// execute beforeEachFeature actions
	ctx = e.processFeatureActions(ctx, t, feature, e.getBeforeFeatureActions())

	// execute feature test
	ctx = e.execFeature(ctx, t, featureName, feature)

	// execute afterEachFeature actions
	return e.processFeatureActions(ctx, t, feature, e.getAfterFeatureActions())
}

// processFeatureActions is used to run a series of feature action that were configured as
// BeforeEachFeature or AfterEachFeature
func (e *testEnv) processFeatureActions(ctx context.Context, t *testing.T, feature types.Feature, actions []action) context.Context {
	t.Helper()
	var err error
	out := ctx
	for _, action := range actions {
		out, err = action.runWithFeature(out, e.cfg, t, deepCopyFeature(feature))
		if err != nil {
			t.Fatalf("%s failure: %s", action.role, err)
		}
	}
	return out
}

// processTests is a wrapper function that can be invoked by either Test or TestInParallel methods.
// Depending on the configuration of if the parallel tests are enabled or not, this will change the
// nature of how the test gets executed.
//
// In case if the parallel run of test features are enabled, this function will invoke the processTestFeature
// as a go-routine to get them to run in parallel
func (e *testEnv) processTests(ctx context.Context, t *testing.T, enableParallelRun bool, testFeatures ...types.Feature) context.Context {
	t.Helper()
	dedicatedTestEnv := newChildTestEnv(e)
	if dedicatedTestEnv.cfg.DryRunMode() {
		klog.V(2).Info("e2e-framework is being run in dry-run mode. This will skip all the before/after step functions configured around your test assessments and features")
	}
	if ctx == nil {
		panic("nil context") // this should never happen
	}
	if len(testFeatures) == 0 {
		t.Log("No test testFeatures provided, skipping test")
		return ctx
	}
	beforeTestActions := dedicatedTestEnv.getBeforeTestActions()
	afterTestActions := dedicatedTestEnv.getAfterTestActions()

	runInParallel := dedicatedTestEnv.cfg.ParallelTestEnabled() && enableParallelRun

	if runInParallel {
		klog.V(4).Info("Running test features in parallel")
	}

	ctx = dedicatedTestEnv.processTestActions(ctx, t, beforeTestActions)

	var wg sync.WaitGroup
	for i, feature := range testFeatures {
		featureTestEnv := newChildTestEnv(dedicatedTestEnv)
		featureCopy := feature
		featName := feature.Name()
		if featName == "" {
			featName = fmt.Sprintf("Feature-%d", i+1)
		}
		if runInParallel {
			wg.Add(1)
			go func(ctx context.Context, w *sync.WaitGroup, featName string, f types.Feature) {
				defer w.Done()
				_ = featureTestEnv.processTestFeature(ctx, t, featName, f)
			}(ctx, &wg, featName, featureCopy)
		} else {
			ctx = featureTestEnv.processTestFeature(ctx, t, featName, featureCopy)
			// In case if the feature under test has failed, skip reset of the features
			// that are part of the same test
			if featureTestEnv.cfg.FailFast() && t.Failed() {
				break
			}
		}
	}
	if runInParallel {
		wg.Wait()
	}
	return dedicatedTestEnv.processTestActions(ctx, t, afterTestActions)
}

// TestInParallel executes a series a feature tests from within a
// TestXXX function in parallel
//
// Feature setups and teardowns are executed at the same *testing.T
// contextual level as the "test" that invoked this method. Assessments
// are executed as a subtests of the feature.  This approach allows
// features/assessments to be filtered using go test -run flag.
//
// Feature tests will have access to and able to update the context
// passed to it.
//
// BeforeTest and AfterTest operations are executed before and after
// the feature is tested respectively.
//
// BeforeTest and AfterTest operations are run in series of the entire
// set of features being passed to this call while the feature themselves
// are executed in parallel to avoid duplication of action that might happen
// in BeforeTest and AfterTest actions
func (e *testEnv) TestInParallel(t *testing.T, testFeatures ...types.Feature) context.Context {
	t.Helper()
	return e.processTests(e.ctx, t, true, testFeatures...)
}

// Test executes a feature test from within a TestXXX function.
//
// Feature setups and teardowns are executed at the same *testing.T
// contextual level as the "test" that invoked this method. Assessments
// are executed as a subtests of the feature.  This approach allows
// features/assessments to be filtered using go test -run flag.
//
// Feature tests will have access to and able to update the context
// passed to it.
//
// BeforeTest and AfterTest operations are executed before and after
// the feature is tested respectively.
func (e *testEnv) Test(t *testing.T, testFeatures ...types.Feature) context.Context {
	t.Helper()
	return e.processTests(e.ctx, t, false, testFeatures...)
}

// Finish registers funcs that are executed at the end of the
// test suite.
func (e *testEnv) Finish(funcs ...Func) types.Environment {
	if len(funcs) == 0 {
		return e
	}

	e.actions = append(e.actions, action{role: roleFinish, funcs: funcs})
	return e
}

// EnvConf returns the test environment's environment configuration
func (e *testEnv) EnvConf() *envconf.Config {
	cfg := *e.cfg
	return &cfg
}

// Run is to launch the test suite from a TestMain function.
// It will run m.Run() and exercise all test functions in the
// package.  This method will all Env.Setup operations prior to
// starting the tests and run all Env.Finish operations after
// before completing the suite.
func (e *testEnv) Run(m *testing.M) (exitCode int) {
	e.panicOnMissingContext()
	ctx := e.ctx

	setups := e.getSetupActions()
	// fail fast on setup, upon err exit
	var err error

	defer func() {
		// Recover and see if the panic handler is disabled. If it is disabled, panic and stop the workflow.
		// Otherwise, log and continue with running the Finish steps of the Test suite
		rErr := recover()
		if rErr != nil {
			if e.cfg.DisableGracefulTeardown() {
				panic(rErr)
			}
			klog.Errorf("Recovering from panic and running finish actions: %s, stack: %s", rErr, string(debug.Stack()))
			// Set this exit code value to non 0 to indicate that the test suite has failed
			// Not doing this will mark the test suite as passed even though there was a panic
			exitCode = 1
		}

		finishes := e.getFinishActions()
		// attempt to gracefully clean up.
		// Upon error, log and continue.
		for _, fin := range finishes {
			// context passed down to each finish step
			if ctx, err = fin.run(ctx, e.cfg); err != nil {
				klog.V(2).ErrorS(err, "Cleanup failed", "action", fin.role)
			}
		}
		e.ctx = ctx
	}()

	for _, setup := range setups {
		// context passed down to each setup
		if ctx, err = setup.run(ctx, e.cfg); err != nil {
			klog.Errorf("%s failure: %s", setup.role, err)
			return 1
		}
	}
	e.ctx = ctx

	// Execute the test suite
	return m.Run()
}

func (e *testEnv) getActionsByRole(r actionRole) []action {
	if e.actions == nil {
		return nil
	}

	var result []action
	for _, a := range e.actions {
		if a.role == r {
			result = append(result, a)
		}
	}

	return result
}

func (e *testEnv) getSetupActions() []action {
	return e.getActionsByRole(roleSetup)
}

func (e *testEnv) getBeforeTestActions() []action {
	return e.getActionsByRole(roleBeforeTest)
}

func (e *testEnv) getBeforeFeatureActions() []action {
	return e.getActionsByRole(roleBeforeFeature)
}

func (e *testEnv) getAfterFeatureActions() []action {
	return e.getActionsByRole(roleAfterFeature)
}

func (e *testEnv) getAfterTestActions() []action {
	return e.getActionsByRole(roleAfterTest)
}

func (e *testEnv) getFinishActions() []action {
	finishAction := e.getActionsByRole(roleFinish)
	if featuregate.DefaultFeatureGate.Enabled(featuregate.ReverseTestFinishExecutionOrder) {
		sort.Slice(finishAction, func(i, j int) bool {
			return i > j
		})
	}
	return finishAction
}

func (e *testEnv) executeSteps(ctx context.Context, t *testing.T, steps []types.Step) context.Context {
	t.Helper()
	if e.cfg.DryRunMode() {
		return ctx
	}
	for _, setup := range steps {
		ctx = setup.Func()(ctx, t, e.cfg)
	}
	return ctx
}

func (e *testEnv) execFeature(ctx context.Context, t *testing.T, featName string, f types.Feature) context.Context {
	t.Helper()
	// feature-level subtest
	t.Run(featName, func(newT *testing.T) {
		newT.Helper()

		if fDescription, ok := f.(types.DescribableFeature); ok && fDescription.Description() != "" {
			t.Logf("Processing Feature: %s", fDescription.Description())
		}

		// setups run at feature-level
		setups := features.GetStepsByLevel(f.Steps(), types.LevelSetup)
		ctx = e.executeSteps(ctx, newT, setups)

		// assessments run as feature/assessment sub level
		assessments := features.GetStepsByLevel(f.Steps(), types.LevelAssess)

		failed := false
		for i, assess := range assessments {
			assessName := assess.Name()
			if dAssess, ok := assess.(types.DescribableStep); ok && dAssess.Description() != "" {
				t.Logf("Processing Assessment: %s", dAssess.Description())
			}
			if assessName == "" {
				assessName = fmt.Sprintf("Assessment-%d", i+1)
			}
			// shouldFailNow catches whether t.FailNow() is called in the assessment.
			// If it is, we won't proceed with the next assessment.
			var shouldFailNow bool
			newT.Run(assessName, func(internalT *testing.T) {
				internalT.Helper()
				skipped, message := e.requireAssessmentProcessing(assess, i+1)
				if skipped {
					internalT.Skip(message)
				}
				// Set shouldFailNow to true before actually running the assessment, because if the assessment
				// calls t.FailNow(), the function will be abruptly stopped in the middle of `e.executeSteps()`.
				shouldFailNow = true
				ctx = e.executeSteps(ctx, internalT, []types.Step{assess})
				// If we reach this point, it means the assessment did not call t.FailNow().
				shouldFailNow = false
			})
			// Check if the Test assessment under question performed either 2 things:
			// - a t.FailNow() invocation
			// - a `t.Fail()` or `t.Failed()` invocation
			// In one of those cases, we need to track that and stop the next set of assessment in the feature
			// under test from getting executed.
			if shouldFailNow || (e.cfg.FailFast() && newT.Failed()) {
				failed = true
				break
			}
		}

		// Let us fail the test fast and not run the teardown in case if the framework specific fail-fast mode is
		// invoked to make sure we leave the traces of the failed test behind to enable better debugging for the
		// test developers
		if e.cfg.FailFast() && failed {
			newT.FailNow()
		}

		// teardowns run at feature-level
		teardowns := features.GetStepsByLevel(f.Steps(), types.LevelTeardown)
		ctx = e.executeSteps(ctx, newT, teardowns)
	})

	return ctx
}

// requireFeatureProcessing is a wrapper around the requireProcessing function to process the feature level validation
func (e *testEnv) requireFeatureProcessing(f types.Feature) (skip bool, message string) {
	requiredRegexp := e.cfg.FeatureRegex()
	skipRegexp := e.cfg.SkipFeatureRegex()
	return e.requireProcessing("feature", f.Name(), requiredRegexp, skipRegexp, f.Labels())
}

// requireAssessmentProcessing is a wrapper around the requireProcessing function to process the Assessment level validation
func (e *testEnv) requireAssessmentProcessing(a types.Step, assessmentIndex int) (skip bool, message string) {
	requiredRegexp := e.cfg.AssessmentRegex()
	skipRegexp := e.cfg.SkipAssessmentRegex()
	assessmentName := a.Name()
	if assessmentName == "" {
		assessmentName = fmt.Sprintf("Assessment-%d", assessmentIndex)
	}
	return e.requireProcessing("assessment", assessmentName, requiredRegexp, skipRegexp, nil)
}

// requireProcessing is a utility function that can be used to make a decision on if a specific Test assessment or feature needs to be
// processed or not.
// testName argument indicate the Feature Name or test Name that can be mapped against the skip or include regex flags
// to decide if the entity in question will need processing.
// This function also perform a label check against include/skip labels to make sure only those features to make sure
// we can filter out all the non-required features during the test execution
func (e *testEnv) requireProcessing(kind, testName string, requiredRegexp, skipRegexp *regexp.Regexp, labels types.Labels) (skip bool, message string) {
	if requiredRegexp != nil && !requiredRegexp.MatchString(testName) {
		skip = true
		message = fmt.Sprintf(`Skipping %s "%s": name not matched`, kind, testName)
		return skip, message
	}
	if skipRegexp != nil && skipRegexp.MatchString(testName) {
		skip = true
		message = fmt.Sprintf(`Skipping %s: "%s": name matched`, kind, testName)
		return skip, message
	}

	if labels != nil {
		// only run a feature if all its label keys and values match those specified
		// with --labels
		matches := 0
		for key, vals := range e.cfg.Labels() {
			for _, v := range vals {
				if labels.Contains(key, v) {
					matches++
					break // continue with next key
				}
			}
		}

		if len(e.cfg.Labels()) != matches {
			skip = true
			var kvs []string
			for k, v := range labels {
				kvs = append(kvs, fmt.Sprintf("%s=%s", k, v)) // prettify output
			}
			message = fmt.Sprintf(`Skipping feature "%s": unmatched labels "%s"`, testName, kvs)
			return skip, message
		}

		// skip running a feature if labels matches with --skip-labels
		for key, vals := range e.cfg.SkipLabels() {
			for _, v := range vals {
				if labels.Contains(key, v) {
					skip = true
					message = fmt.Sprintf(`Skipping feature "%s": matched label provided in --skip-lables "%s=%s"`, testName, key, labels[key])
					return skip, message
				}
			}
		}
	}
	return skip, message
}

// deepCopyConfig just copies the values from the Config to create a deep
// copy to avoid mutation when we just want an informational copy.
func (e *testEnv) deepCopyConfig() *envconf.Config {
	// Basic copy which takes care of all the basic types (str, bool...)
	configCopy := *e.cfg

	// Manually setting fields that are struct types
	if client := e.cfg.GetClient(); client != nil {
		// Need to recreate the underlying client because client.Resource is not thread safe
		// Panic on error because this should never happen since the client was built once already
		clientCopy, err := klient.New(client.RESTConfig())
		if err != nil {
			panic(err)
		}
		configCopy.WithClient(clientCopy)
	}
	if e.cfg.AssessmentRegex() != nil {
		configCopy.WithAssessmentRegex(e.cfg.AssessmentRegex().String())
	}
	if e.cfg.FeatureRegex() != nil {
		configCopy.WithFeatureRegex(e.cfg.FeatureRegex().String())
	}
	if e.cfg.SkipAssessmentRegex() != nil {
		configCopy.WithSkipAssessmentRegex(e.cfg.SkipAssessmentRegex().String())
	}
	if e.cfg.SkipFeatureRegex() != nil {
		configCopy.WithSkipFeatureRegex(e.cfg.SkipFeatureRegex().String())
	}

	labels := make(map[string][]string, len(e.cfg.Labels()))
	for k, vals := range e.cfg.Labels() {
		copyVals := make([]string, len(vals))
		copyVals = append(copyVals, vals...)
		labels[k] = copyVals
	}
	configCopy.WithLabels(labels)

	skipLabels := make(map[string][]string, len(e.cfg.SkipLabels()))
	for k, vals := range e.cfg.SkipLabels() {
		copyVals := make([]string, len(vals))
		copyVals = append(copyVals, vals...)
		skipLabels[k] = copyVals
	}
	configCopy.WithSkipLabels(e.cfg.SkipLabels())
	return &configCopy
}

// deepCopyFeature just copies the values from the Feature to create a deep
// copy to avoid mutation when we just want an informational copy.
func deepCopyFeature(f types.Feature) types.Feature {
	fcopy := features.New(f.Name())
	for k, vals := range f.Labels() {
		for _, v := range vals {
			fcopy = fcopy.WithLabel(k, v)
		}
	}
	f.Steps()
	for _, step := range f.Steps() {
		fcopy = fcopy.WithStep(step.Name(), step.Level(), nil)
	}
	return fcopy.Feature()
}

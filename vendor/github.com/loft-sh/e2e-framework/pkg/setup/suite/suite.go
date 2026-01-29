package suite

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

var (
	dependencies = map[string]*dependency{}
)

type SetupContextCallback func(context.Context) context.Context

type ContextCallback func(context.Context)

type Dependency interface {
	Label() string
	IsFocused() bool
	Dependencies() []Dependency
	Setup(context context.Context) (context.Context, error)
	Import(context context.Context) (context.Context, error)
	Teardown(context context.Context) (context.Context, error)
}

type dependency struct {
	label      string
	setupFn    setup.Func
	importFn   setup.Func
	teardownFn setup.Func

	beforeEach SetupContextCallback
	beforeAll  SetupContextCallback
	afterEach  ContextCallback
	afterAll   ContextCallback

	dependencies []Dependency
}

func (d *dependency) Label() string {
	return d.label
}

func (d *dependency) IsFocused() bool {
	return IsFocused(d.label)
}

func (d *dependency) Dependencies() []Dependency {
	return d.dependencies
}

func (d *dependency) Setup(ctx context.Context) (context.Context, error) {
	if IsFocused(d.label) {
		return d.setupFn(ctx)
	}

	return ctx, nil
}

func (d *dependency) Import(ctx context.Context) (context.Context, error) {
	if IsFocused(d.label) {
		return d.importFn(ctx)
	}

	return ctx, nil
}

func (d *dependency) Teardown(ctx context.Context) (context.Context, error) {
	if IsFocused(d.label) {
		return d.teardownFn(ctx)
	}

	return ctx, nil
}

func Lookup(label string) (Dependency, bool) {
	dep, ok := dependencies[label]
	return dep, ok
}

type Options func(d *dependency) Dependency

func WithLabel(label string) Options {
	return func(d *dependency) Dependency {
		d.label = label
		return d
	}
}

func WithSetup(setupFn setup.Func) Options {
	return func(d *dependency) Dependency {
		d.setupFn = setupFn
		return d
	}
}

func WithImport(importFn setup.Func) Options {
	return func(d *dependency) Dependency {
		d.importFn = importFn
		return d
	}
}

func WithTeardown(teardownFn setup.Func) Options {
	return func(d *dependency) Dependency {
		d.teardownFn = teardownFn
		return d
	}
}

func WithDependencies(dependencies ...Dependency) Options {
	return func(d *dependency) Dependency {
		d.dependencies = dependencies
		return d
	}
}

func WithBeforeEach(fn SetupContextCallback) Options {
	return func(d *dependency) Dependency {
		d.beforeEach = fn
		return d
	}
}

func WithBeforeAll(fn SetupContextCallback) Options {
	return func(d *dependency) Dependency {
		d.beforeAll = fn
		return d
	}
}

func WithAfterEach(fn ContextCallback) Options {
	return func(d *dependency) Dependency {
		d.afterEach = fn
		return d
	}
}

func WithAfterAll(fn ContextCallback) Options {
	return func(d *dependency) Dependency {
		d.afterAll = fn
		return d
	}
}

func AddDependency(dep Dependency) Options {
	return func(d *dependency) Dependency {
		d.dependencies = append(d.dependencies, dep)
		return d
	}
}

func Define(opts ...Options) Dependency {
	d := &dependency{}
	for _, opt := range opts {
		opt(d)
	}
	if dependencies[d.label] != nil {
		panic("dependency already defined")
	}
	dependencies[d.label] = d
	return d
}

func Use(dep Dependency) Labels {
	depLabels := []string{dep.Label()}
	for _, dep := range dep.Dependencies() {
		depLabels = append(depLabels, dep.Label())
	}
	return depLabels
}

func NodeTransformer(nodeType types.NodeType, _ Offset, text string, args []any) (string, []any, []error) {
	var newArgs []any

	var (
		deps    []*dependency
		body    func()
		ordered = false
	)
	for _, arg := range args {
		if arg == Ordered {
			ordered = true
			newArgs = append(newArgs, arg)
			continue
		}

		switch x := arg.(type) {
		case func():
			body = x
		case Labels:
			for _, label := range x {
				if dep, ok := dependencies[label]; ok {
					deps = append(deps, dep)
				}
			}
			newArgs = append(newArgs, x)
		default:
			newArgs = append(newArgs, x)
		}
	}

	var (
		beforeEach []SetupContextCallback
		afterEach  []ContextCallback
		beforeAll  []SetupContextCallback
		afterAll   []ContextCallback
	)
	for _, dep := range deps {
		if dep.beforeEach != nil {
			beforeEach = append(beforeEach, dep.beforeEach)
		}
		if dep.afterEach != nil {
			afterEach = append(afterEach, dep.afterEach)
		}
		if dep.beforeAll != nil {
			beforeAll = append(beforeAll, dep.beforeAll)
		}
		if dep.afterAll != nil {
			afterAll = append(afterAll, dep.afterAll)
		}
	}

	if nodeType == types.NodeTypeContainer {
		newArgs = append(newArgs, func() {
			if ordered {
				if len(beforeAll) > 0 {
					BeforeAll(func(ctx context.Context) context.Context {
						first, rest := beforeAll[0], beforeAll[1:]
						for _, before := range rest {
							ctx = before(ctx)
						}
						ctx = first(ctx)
						return ctx
					})
				}

				if len(afterAll) > 0 {
					AfterAll(func(ctx context.Context) {
						first, rest := afterAll[0], afterAll[1:]
						for _, after := range rest {
							after(ctx)
						}
						first(ctx)
					})
				}
			} else {
				if len(beforeEach) > 0 {
					BeforeEach(func(ctx context.Context) context.Context {
						first, rest := beforeEach[0], beforeEach[1:]
						for _, before := range rest {
							ctx = before(ctx)
						}
						ctx = first(ctx)
						return ctx
					})
				}

				if len(afterEach) > 0 {
					AfterEach(func(ctx context.Context) {
						first, rest := afterEach[0], afterEach[1:]
						for _, after := range rest {
							after(ctx)
						}
						first(ctx)
					})
				}
			}
			body()
		})
	} else if body != nil {
		newArgs = append(newArgs, body)
	}

	return text, newArgs, nil
}

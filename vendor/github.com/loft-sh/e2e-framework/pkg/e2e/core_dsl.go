package e2e

import (
	"cmp"
	"context"
	"reflect"
	"sync"

	e2econtext "github.com/loft-sh/e2e-framework/pkg/context"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

var (
	setupOnly    = false
	teardown     = true
	teardownOnly = false
)

func SetSetupOnly(b bool) {
	setupOnly = b
}

func SetTeardown(b bool) {
	teardown = b
}

func SetTeardownOnly(b bool) {
	teardownOnly = b
}

type ContextMiddleware func(context.Context) context.Context

type key int

const parentKey key = iota

type contextStack struct {
	lock   sync.Mutex
	cur    context.Context
	prev   context.Context
	parent context.Context
}

func newContextStack() *contextStack {
	return &contextStack{
		cur: context.TODO(),
	}
}

func (cs *contextStack) pushContext(ctx context.Context) context.Context {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	parentCtx := cs.cur

	// Copy values down from parent
	if parentCtx != nil {
		cs.cur = e2econtext.WithValues(
			context.WithValue(ctx, parentKey, parentCtx),
			parentCtx,
		)
	}

	return cs.cur
}

func (cs *contextStack) popContext(ctx context.Context) context.Context {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.prev = cs.cur
	if parentCtx := ctx.Value(parentKey); parentCtx != nil {
		cs.cur = parentCtx.(context.Context)
	}
	return cs.prev
}

var suiteContextStack = newContextStack()

func ContextualAroundNode(ctx context.Context) context.Context {
	report := ginkgo.CurrentSpecReport()
	events := report.SpecEvents.WithType(types.SpecEventNodeStart)
	event := events[len(events)-1]

	switch event.NodeType {
	case types.NodeTypeCleanupAfterEach:
		fallthrough
	case types.NodeTypeCleanupAfterSuite:
		fallthrough
	case types.NodeTypeCleanupAfterAll:
		fallthrough
	case types.NodeTypeReportAfterSuite:
		fallthrough
	case types.NodeTypeCleanupInvalid:
		return e2econtext.WithValues(ctx, cmp.Or(suiteContextStack.prev, suiteContextStack.cur))
	default:
		return e2econtext.WithValues(ctx, suiteContextStack.cur)
	}
}

func ContextualNodeTransformer(nodeType types.NodeType, _ ginkgo.Offset, text string, args []any) (string, []any, []error) {
	var newArgs []any

	for _, arg := range args {
		isFn := reflect.TypeOf(arg).Kind() == reflect.Func
		if !isFn {
			newArgs = append(newArgs, arg)
			continue
		}

		switch x := arg.(type) {
		case func(context.Context) context.Context:
			newArgs = append(newArgs, func(ctx context.Context) {
				ctx = suiteContextStack.pushContext(getFnCtxCtxCallback(nodeType, x)(ctx))
				defer ginkgo.DeferCleanup(func() {
					suiteContextStack.popContext(ctx)
				})
			})
		case func(context.Context):
			newArgs = append(newArgs, getFnCtxCallback(nodeType, x))
		case func():
			newArgs = append(newArgs, getFnCallback(nodeType, x))
		default:
			newArgs = append(newArgs, arg)
		}
	}

	return text, newArgs, nil
}

func getFnCallback(nodeType types.NodeType, fn func()) func() {
	switch nodeType {
	case types.NodeTypeBeforeEach:
		return func() {
			if setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn()
			}
		}
	case types.NodeTypeAfterSuite:
		fallthrough
	case types.NodeTypeAfterAll:
		return func() {
			if !teardown || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn()
			}
		}
	case types.NodeTypeAfterEach:
		fallthrough
	case types.NodeTypeIt:
		return func() {
			if teardownOnly || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn()
			}
		}
	case types.NodeTypeContainer:
		return func() {
			if setupOnly {
				// Insert a case to focus so all other specs are skipped.
				ginkgo.It("[Setup]", ginkgo.Focus, func() {
					ginkgo.By("Setting up environment")
				})
			}

			if teardownOnly {
				// Insert a case to focus so all other specs are skipped.
				ginkgo.It("[Teardown]", ginkgo.Focus, func() {
					ginkgo.By("Tearing down environment")
				})
			}

			fn()
		}
	}
	return fn
}

func getFnCtxCallback(nodeType types.NodeType, fn func(context.Context)) func(context.Context) {
	switch nodeType {
	case types.NodeTypeBeforeEach:
		return func(ctx context.Context) {
			if setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn(ctx)
			}
		}
	case types.NodeTypeAfterSuite:
		fallthrough
	case types.NodeTypeAfterAll:
		return func(ctx context.Context) {
			if !teardown || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn(ctx)
			}
		}
	case types.NodeTypeAfterEach:
		fallthrough
	case types.NodeTypeIt:
		return func(ctx context.Context) {
			if teardownOnly || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
			} else {
				fn(ctx)
			}
		}
	}
	return fn
}

func getFnCtxCtxCallback(nodeType types.NodeType, fn func(context.Context) context.Context) func(context.Context) context.Context {
	switch nodeType {
	case types.NodeTypeBeforeEach:
		return func(ctx context.Context) context.Context {
			if setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
				return ctx
			} else {
				return fn(ctx)
			}
		}
	case types.NodeTypeAfterSuite:
		fallthrough
	case types.NodeTypeAfterAll:
		return func(ctx context.Context) context.Context {
			if !teardown || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
				return ctx
			} else {
				return fn(ctx)
			}
		}
	case types.NodeTypeAfterEach:
		fallthrough
	case types.NodeTypeIt:
		return func(ctx context.Context) context.Context {
			if teardownOnly || setupOnly {
				ginkgo.By("[" + nodeType.String() + "] skipped")
				return ctx
			} else {
				return fn(ctx)
			}
		}
	}
	return fn
}

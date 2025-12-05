package e2e

import (
	"cmp"
	"context"
	"reflect"

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

var suiteContextStack = NewStack[context.Context]()

var process1ContextStack = NewStack[context.Context]()

func ContextualAroundNode(ctx context.Context) context.Context {
	report := ginkgo.CurrentSpecReport()
	events := report.SpecEvents.WithType(types.SpecEventNodeStart)
	event := events[len(events)-1]

	switch event.NodeType {
	case types.NodeTypeSynchronizedAfterSuite:
		return ctx
	case types.NodeTypeCleanupAfterEach:
		fallthrough
	case types.NodeTypeCleanupAfterSuite:
		fallthrough
	case types.NodeTypeCleanupAfterAll:
		fallthrough
	case types.NodeTypeReportAfterSuite:
		fallthrough
	case types.NodeTypeCleanupInvalid:
		last, _ := suiteContextStack.Last()
		current, _ := suiteContextStack.Peek()
		return e2econtext.WithValues(ctx, cmp.Or(last, current, context.TODO()))
	default:
		current, _ := suiteContextStack.Peek()
		return e2econtext.WithValues(ctx, cmp.Or(current, context.TODO()))
	}
}

func ContextualNodeTransformer(nodeType types.NodeType, _ ginkgo.Offset, text string, args []any) (string, []any, []error) {
	var newArgs []any

	for idx, arg := range args {
		isFn := reflect.TypeOf(arg).Kind() == reflect.Func
		if !isFn {
			newArgs = append(newArgs, arg)
			continue
		}

		switch x := arg.(type) {
		case func(context.Context) context.Context:
			contextStack := suiteContextStack
			if nodeType == types.NodeTypeSynchronizedBeforeSuite && idx == 0 {
				contextStack = process1ContextStack
			}
			newArgs = append(newArgs, func(ctx context.Context) {
				parentCtx, _ := suiteContextStack.Peek()
				if parentCtx != nil {
					ctx = e2econtext.WithValues(ctx, parentCtx)
				}
				contextStack.Push(getFnCtxCtxCallback(nodeType, x)(ctx))
				ginkgo.DeferCleanup(func() {
					contextStack.Pop()
				})
			})
		case func(context.Context) (context.Context, []byte):
			contextStack := suiteContextStack
			if nodeType == types.NodeTypeSynchronizedBeforeSuite && idx == 0 {
				contextStack = process1ContextStack
			}
			newArgs = append(newArgs, func(ctx context.Context) []byte {
				parentCtx, _ := suiteContextStack.Peek()
				if parentCtx != nil {
					ctx = e2econtext.WithValues(ctx, parentCtx)
				}
				fnCtx, data := x(ctx)
				contextStack.Push(fnCtx)
				ginkgo.DeferCleanup(func() {
					contextStack.Pop()
				})

				return data
			})
		case func(context.Context, []byte) context.Context:
			contextStack := suiteContextStack
			if nodeType == types.NodeTypeSynchronizedBeforeSuite && idx == 0 {
				contextStack = process1ContextStack
			}
			newArgs = append(newArgs, func(ctx context.Context, data []byte) {
				parentCtx, _ := suiteContextStack.Peek()
				if parentCtx != nil {
					ctx = e2econtext.WithValues(ctx, parentCtx)
				}
				contextStack.Push(x(ctx, data))
				ginkgo.DeferCleanup(func() {
					contextStack.Pop()
				})
			})
		case func(context.Context):
			fn := getFnCtxCallback(nodeType, x)
			if nodeType == types.NodeTypeSynchronizedAfterSuite {
				newArgs = append(newArgs, func(ctx context.Context) {
					contextStack := suiteContextStack
					if idx == 1 {
						contextStack = process1ContextStack
					}
					last, _ := contextStack.Last()
					current, _ := contextStack.Peek()
					fn(e2econtext.WithValues(ctx, cmp.Or(last, current, context.TODO())))
				})
			} else {
				newArgs = append(newArgs, fn)
			}
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
	case types.NodeTypeSynchronizedAfterSuite:
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
	case types.NodeTypeSynchronizedAfterSuite:
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

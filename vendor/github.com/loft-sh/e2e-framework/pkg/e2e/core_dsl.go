package e2e

import (
	"context"
	"reflect"
	"sync"

	e2econtext "github.com/loft-sh/e2e-framework/pkg/context"
	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/onsi/ginkgo/v2"
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

type contextStack struct {
	lock *sync.Mutex
	cur  context.Context
}

func newContextStack() *contextStack {
	return &contextStack{
		lock: &sync.Mutex{},
		cur:  context.TODO(),
	}
}

func (cs *contextStack) push(fn ContextMiddleware) func(specContext ginkgo.SpecContext) {
	return func(specContext ginkgo.SpecContext) {
		cs.lock.Lock()
		defer cs.lock.Unlock()

		ctx := context.TODO()
		parentCtx := cs.cur

		// Copy values down from parent
		if parentCtx != nil {
			ctx = e2econtext.WithValues(ctx, parentCtx)
		}

		// Call function with extended spec context
		cs.cur = e2econtext.WithValues(specContext, ctx)
		cs.cur = fn(cs.cur)

		ginkgo.DeferCleanup(func() {
			cs.lock.Lock()
			defer cs.lock.Unlock()

			cs.cur = parentCtx
		})
	}
}

var suiteContextStack = newContextStack()

func BeforeSuite(body any, args ...any) bool {
	ginkgo.GinkgoHelper()

	combinedArgs := []any{body}
	combinedArgs = append(combinedArgs, args...)

	ctxMiddleware, remainder := splitContextMiddlewareArg(combinedArgs)
	if ctxMiddleware != nil {
		return ginkgo.BeforeSuite(suiteContextStack.push(ctxMiddleware), remainder...)
	}
	return ginkgo.BeforeSuite(body, args...)
}

func AfterSuite(body any, args ...any) bool {
	ginkgo.GinkgoHelper()

	combinedArgs := []any{body}
	combinedArgs = append(combinedArgs, args...)

	bodyFn, remainder := splitBodyFunctionArg(combinedArgs)
	return ginkgo.AfterSuite(func(specContext ginkgo.SpecContext) {
		ctx := e2econtext.WithValues(specContext, suiteContextStack.cur)
		if !teardown {
			ginkgo.By("[AfterSuite] disabled")
			return
		}
		bodyFn(ctx)
	}, remainder...)
}

var Context = Describe

func Describe(text string, args ...any) bool {
	ginkgo.GinkgoHelper()

	bodyFn, remainder := splitBodyFunctionArg(args)
	return ginkgo.Describe(text, append(remainder, ginkgo.Ordered, func() {
		bodyFn(suiteContextStack.cur)
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
	}))
}

func BeforeAll(args ...any) bool {
	ginkgo.GinkgoHelper()

	ctxMiddleware, remainder := splitContextMiddlewareArg(args)
	if ctxMiddleware != nil {
		return ginkgo.BeforeAll(append(remainder, suiteContextStack.push(ctxMiddleware)))
	}

	bodyFn, remainder := splitBodyFunctionArg(args)
	return ginkgo.BeforeAll(append(remainder, func(specContext ginkgo.SpecContext) {
		bodyFn(e2econtext.WithValues(specContext, suiteContextStack.cur))
	}))
}

func BeforeEach(args ...any) bool {
	ginkgo.GinkgoHelper()

	if setupOnly {
		return true
	}

	ctxMiddleware, remainder := splitContextMiddlewareArg(args)
	bodyFn, remainder := splitBodyFunctionArg(remainder)
	if ctxMiddleware != nil {
		return ginkgo.BeforeEach(append(remainder, suiteContextStack.push(ctxMiddleware)))
	}

	return ginkgo.BeforeEach(append(remainder, func(specContext ginkgo.SpecContext) {
		bodyFn(e2econtext.WithValues(specContext, suiteContextStack.cur))
	}))
}

func AfterEach(args ...any) bool {
	ginkgo.GinkgoHelper()

	if teardownOnly {
		return true
	}

	bodyFn, remainder := splitBodyFunctionArg(args)
	return ginkgo.AfterEach(append(remainder, func(specContext ginkgo.SpecContext) {
		bodyFn(e2econtext.WithValues(specContext, suiteContextStack.cur))
	}))
}

func AfterAll(args ...any) bool {
	ginkgo.GinkgoHelper()

	bodyFn, remainder := splitBodyFunctionArg(args)
	return ginkgo.AfterAll(append(remainder, func(specContext ginkgo.SpecContext) {
		ctx := e2econtext.WithValues(specContext, suiteContextStack.cur)
		if !teardown {
			ginkgo.By("[AfterAll] disabled")
			return
		}
		bodyFn(ctx)
	}))
}

func It(text string, args ...any) bool {
	ginkgo.GinkgoHelper()

	if setupOnly || teardownOnly {
		// Skip adding test cases... a placeholder one will be used instead.
		return true
	}

	bodyFn, remainder := splitBodyFunctionArg(args)
	return ginkgo.It(text, append(remainder, func(specContext ginkgo.SpecContext) {
		bodyFn(e2econtext.WithValues(specContext, suiteContextStack.cur))
	}))
}

func DeferCleanup(args ...any) {
	ginkgo.GinkgoHelper()

	if setupFunc, ok := extractSetupFunc(args); ok {
		if deferCtx, ok := extractDeferCtx(args); ok {
			// Should use the passed ctx
			if e2econtext.IsComposed(deferCtx) {
				ginkgo.DeferCleanup(setupFunc, deferCtx)
			} else {
				ginkgo.DeferCleanup(func(ctx context.Context) error {
					_, err := setupFunc(e2econtext.WithValues(deferCtx, suiteContextStack.cur))
					return err
				})
			}
		} else {
			// Allow ginkgo to pass a SpecContext, but compose it to add values
			ginkgo.DeferCleanup(func(specContext ginkgo.SpecContext) error {
				_, err := setupFunc(e2econtext.WithValues(specContext, suiteContextStack.cur))
				return err
			})
		}

		return
	}

	ginkgo.DeferCleanup(args...)
}

var contextMiddlewareType = reflect.TypeOf(ContextMiddleware(nil))

func splitContextMiddlewareArg(args []any) (ContextMiddleware, []any) {
	var ctxMiddleware ContextMiddleware
	var remainder []any

	for _, arg := range args {
		argType := reflect.TypeOf(arg)
		if argType.AssignableTo(contextMiddlewareType) {
			ctxMiddleware = arg.(func(ctx context.Context) context.Context)
		} else {
			remainder = append(remainder, arg)
		}
	}

	return ctxMiddleware, remainder
}

func splitBodyFunctionArg(args []any) (func(ctx context.Context), []any) {
	var fn func(context.Context)
	var remainder []any

	for _, arg := range args {
		switch t := reflect.TypeOf(arg); {
		case t.Kind() == reflect.Func:
			if bodyFn := extractBodyFunction(arg); bodyFn != nil {
				fn = bodyFn
				continue
			}
		default:
			remainder = append(remainder, arg)
		}
	}

	return fn, remainder
}

var contextType = reflect.TypeOf(new(context.Context)).Elem()
var specContextType = reflect.TypeOf(new(ginkgo.SpecContext)).Elem()

func extractBodyFunction(arg any) func(specContext context.Context) {
	t := reflect.TypeOf(arg)
	if t.NumOut() > 0 || t.NumIn() > 1 {
		return nil
	}
	if t.NumIn() == 1 {
		if t.In(0).Implements(specContextType) {
			return arg.(func(context.Context))
		} else if t.In(0).Implements(contextType) {
			return arg.(func(context.Context))
		}

		return nil
	}

	body := arg.(func())
	return func(context.Context) { body() }
}

var setupFuncType = reflect.TypeOf(setup.Func(nil))

func extractSetupFunc(args []any) (setup.Func, bool) {
	if len(args) == 0 {
		return nil, false
	}

	firstArg := args[0]
	if firstArg == nil {
		return nil, false
	}

	argType := reflect.TypeOf(firstArg)
	if argType.AssignableTo(setupFuncType) {
		return firstArg.(setup.Func), true
	}

	return nil, false
}

func extractDeferCtx(args []any) (context.Context, bool) {
	if len(args) < 2 {
		return nil, false
	}

	secondArg := args[1]
	if secondArg == nil {
		return nil, false
	}

	if reflect.TypeOf(secondArg).Implements(contextType) {
		return secondArg.(context.Context), true
	}

	return nil, false
}

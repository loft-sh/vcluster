package context

import "context"

func WithValues(ctx, values context.Context) context.Context {
	return &composed{Context: ctx, nextValues: values}
}

type composed struct {
	context.Context
	nextValues context.Context
}

func (c *composed) Value(key interface{}) interface{} {
	if v := c.Context.Value(key); v != nil {
		return v
	}
	return c.nextValues.Value(key)
}

func IsComposed(ctx context.Context) bool {
	_, ok := ctx.(*composed)
	return ok
}

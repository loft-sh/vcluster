package setup

import "context"

type Func func(ctx context.Context) (context.Context, error)

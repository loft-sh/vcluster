package option

type Function[T any] func(*T)

func Resolve[T any](opts ...Function[T]) *T {
	o := new(T)
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

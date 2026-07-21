package telemetry

import "sync"

type cachedValue[T any] struct {
	value     T
	m         sync.Mutex
	retrieved bool
}

func (c *cachedValue[T]) Get(retrieve func() (T, error)) (T, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.retrieved {
		return c.value, nil
	}

	v, err := retrieve()
	if err != nil {
		return v, err
	}

	c.value = v
	c.retrieved = true
	return v, nil
}

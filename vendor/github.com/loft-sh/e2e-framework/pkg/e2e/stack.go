package e2e

// Stack is a generic stack implementation using a slice.
type Stack[T any] struct {
	items          []T
	last           *T
	generation     uint64
	lastGeneration uint64
}

// NewStack creates a new empty stack.
func NewStack[T any]() *Stack[T] {
	return &Stack[T]{
		items: make([]T, 0),
	}
}

// Push adds an item to the top of the stack.
func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

// Pop removes and returns the top item from the stack.
// Returns the zero value of T and false if the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}
	idx := len(s.items) - 1
	item := s.items[idx]
	s.items = s.items[:idx]
	s.last = &item
	s.lastGeneration = s.generation
	return item, true
}

// Peek returns the top item without removing it.
// Returns the zero value of T and false if the stack is empty.
func (s *Stack[T]) Peek() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}
	item := s.items[len(s.items)-1]
	return item, true
}

// Last returns the most recently popped item.
// Returns the zero value of T and false if no item has been popped yet or if
// the last popped item was invalidated by NextGeneration.
func (s *Stack[T]) Last() (T, bool) {
	if s.last == nil || s.lastGeneration != s.generation {
		var zero T
		return zero, false
	}
	return *s.last, true
}

// NextGeneration invalidates the most recently popped item so that Last no
// longer returns it. Callers use this to mark a boundary (e.g. a new spec)
// across which previously popped items must not be observable.
func (s *Stack[T]) NextGeneration() {
	s.generation++
}

// IsEmpty returns true if the stack is empty.
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Size returns the number of items in the stack.
func (s *Stack[T]) Size() int {
	return len(s.items)
}

// Clear removes all items from the stack.
func (s *Stack[T]) Clear() {
	s.items = s.items[:0]
	s.last = nil
}

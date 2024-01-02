package stackcoll

import (
	"errors"

	"github.com/inoxlang/inox/internal/core"
)

const (
	stackShrinkDivider       = 2
	minShrinkableStackLength = 10 * stackShrinkDivider
	stackInitialSizeFactor   = 2
)

var (
	ErrCannotPopEmptyStack          = errors.New("cannot pop empty stack")
	ErrCannotGetTopElemOfEmptyStack = errors.New("cannot get top element of empty stack")
)

type Stack struct {
	elements []core.Value
}

func NewStack(ctx *core.Context, elements core.Iterable) *Stack {
	stack := &Stack{}

	it := elements.Iterator(ctx, core.IteratorConfiguration{})
	for it.Next(ctx) {
		e := it.Value(ctx)
		stack.elements = append(stack.elements, e)
	}

	return stack
}

func (s *Stack) Push(ctx *core.Context, elems ...core.Value) {
	s.elements = append(s.elements, elems...)
}

func (s *Stack) Pop(ctx *core.Context) {
	if len(s.elements) == 0 {
		panic(ErrCannotPopEmptyStack)
	}
	s.elements = s.elements[:len(s.elements)-1]

	//if the number of elements is too small compared to the capacity of the underlying slice, we shrink the slice
	if len(s.elements) >= minShrinkableStackLength && len(s.elements) <= cap(s.elements)/stackShrinkDivider {
		newSlice := make([]core.Value, len(s.elements))
		copy(newSlice, s.elements)
		s.elements = newSlice
	}
}

func (s *Stack) Peek(ctx *core.Context) core.Value {
	if len(s.elements) == 0 {
		panic(ErrCannotPopEmptyStack)
	}
	return s.elements[len(s.elements)-1]
}

package core

import (
	"fmt"

	"github.com/inoxlang/inox/internal/core/symbolic"
)

func init() {
	RegisterSymbolicGoFunctions([]any{
		Filter, func(ctx *symbolic.Context, iterable symbolic.Iterable, cond symbolic.Value) *symbolic.List {
			return symbolic.NewListOf(symbolic.ANY_SERIALIZABLE)
		},
		GetAtMost, func(ctx *symbolic.Context, maxCount *symbolic.Int, iterable symbolic.SerializableIterable) *symbolic.List {
			element, ok := symbolic.AsSerializable(iterable.IteratorElementValue()).(symbolic.Serializable)
			if !ok {
				element = symbolic.ANY_SERIALIZABLE
				ctx.AddSymbolicGoFunctionError("elements of the iterable are not serializable")
			}
			return symbolic.NewListOf(element)
		},
		Some, func(ctx *symbolic.Context, iterable symbolic.Iterable, cond symbolic.Value) *symbolic.Bool {
			return symbolic.ANY_BOOL
		},
		All, func(ctx *symbolic.Context, iterable symbolic.Iterable, cond symbolic.Value) *symbolic.Bool {
			return symbolic.ANY_BOOL
		},
		None, func(ctx *symbolic.Context, iterable symbolic.Iterable, cond symbolic.Value) *symbolic.Bool {
			return symbolic.ANY_BOOL
		},
	})

}

func Filter(ctx *Context, iterable Iterable, condition Value) *List {
	result := ValueList{}

	switch fil := condition.(type) {
	case AstNode:
		state := ctx.GetClosestState()
		treeWalkState := NewTreeWalkStateWithGlobal(state)

		treeWalkState.PushScope()
		defer treeWalkState.PopScope()

		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			treeWalkState.CurrentLocalScope()[""] = e
			res, err := TreeWalkEval(fil.Node, treeWalkState)
			if err != nil {
				panic(err)
			}
			if res.(Bool) {
				result.elements = append(result.elements, e.(Serializable))
			}
		}
	case Pattern:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			if fil.Test(ctx, e) {
				result.elements = append(result.elements, e.(Serializable))
			}
		}
	default:
		panic(fmt.Errorf("invalid filter : type is %T", fil))
	}

	return WrapUnderlyingList(&result)
}

func GetAtMost(ctx *Context, maxCount Int, iterable SerializableIterable) *List {
	var elements []Serializable
	count := 0

	if indexable, ok := iterable.(Indexable); ok {
		end := min(int(maxCount), indexable.Len())
		for i := 0; i < end; i++ {
			elements = append(elements, indexable.At(ctx, i).(Serializable))
		}
	} else {
		it := iterable.Iterator(ctx, IteratorConfiguration{
			KeysNeverRead: true,
		})
		for count < int(maxCount) && it.Next(ctx) {
			elements = append(elements, it.Value(ctx).(Serializable))
			count++
		}
	}

	return NewWrappedValueListFrom(elements)
}

func Some(ctx *Context, iterable Iterable, condition Value) Bool {

	state := ctx.GetClosestState()
	treeWalkState := NewTreeWalkStateWithGlobal(state)

	treeWalkState.PushScope()
	defer treeWalkState.PopScope()

	switch cond := condition.(type) {
	case AstNode:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			treeWalkState.CurrentLocalScope()[""] = e
			res, err := TreeWalkEval(cond.Node, treeWalkState)
			if err != nil {
				panic(err)
			}
			if res.(Bool) {
				return true
			}
		}
	case Pattern:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			if cond.Test(ctx, e) {
				return true
			}
		}
	}

	return true
}

func All(ctx *Context, iterable Iterable, condition Value) Bool {

	state := ctx.GetClosestState()
	treeWalkState := NewTreeWalkStateWithGlobal(state)

	treeWalkState.PushScope()
	defer treeWalkState.PopScope()

	switch cond := condition.(type) {
	case AstNode:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)

			treeWalkState.CurrentLocalScope()[""] = e
			res, err := TreeWalkEval(cond.Node, treeWalkState)
			if err != nil {
				panic(err)
			}
			if !res.(Bool) {
				return false
			}
		}
	case Pattern:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			if !cond.Test(ctx, e) {
				return false
			}
		}
	}

	return true
}

func None(ctx *Context, iterable Iterable, condition Value) Bool {

	state := ctx.GetClosestState()
	treeWalkState := NewTreeWalkStateWithGlobal(state)

	treeWalkState.PushScope()
	defer treeWalkState.PopScope()

	switch cond := condition.(type) {
	case AstNode:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			treeWalkState.CurrentLocalScope()[""] = e
			res, err := TreeWalkEval(cond.Node, treeWalkState)
			if err != nil {
				panic(err)
			}
			if res.(Bool) {
				return false
			}
		}
	case Pattern:
		it := iterable.Iterator(ctx, IteratorConfiguration{})
		for it.Next(ctx) {
			e := it.Value(ctx)
			if cond.Test(ctx, e) {
				return false
			}
		}
	}

	return true
}

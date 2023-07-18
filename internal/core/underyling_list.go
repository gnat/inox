package core

import (
	"errors"

	"github.com/bits-and-blooms/bitset"
)

type underylingList interface {
	PseudoClonable
	MutableLengthSequence
	Iterable
	ContainsSimple(ctx *Context, v Serializable) bool
	append(ctx *Context, values ...Serializable)
}

// ValueList implements underylingList
type ValueList struct {
	elements     []Serializable
	constraintId ConstraintId
}

func NewWrappedValueList(elements ...Serializable) *List {
	return newList(&ValueList{elements: elements})
}

func NewWrappedValueListFrom(elements []Serializable) *List {
	return newList(&ValueList{elements: elements})
}

func newValueList(elements ...Serializable) *ValueList {
	return &ValueList{elements: elements}
}

func (list *ValueList) ContainsSimple(ctx *Context, v Serializable) bool {
	if !IsSimpleInoxVal(v) {
		panic("only simple values are expected")
	}

	for _, e := range list.elements {
		if v.Equal(nil, e, map[uintptr]uintptr{}, 0) {
			return true
		}
	}
	return false
}

func (list *ValueList) set(ctx *Context, i int, v Value) {
	list.elements[i] = v.(Serializable)
}

func (list *ValueList) SetSlice(ctx *Context, start, end int, seq Sequence) {
	if seq.Len() != end-start {
		panic(errors.New(FormatIndexableShouldHaveLen(end - start)))
	}

	for i := start; i < end; i++ {
		list.elements[i] = seq.At(ctx, i-start).(Serializable)
	}
}

func (list *ValueList) slice(start, end int) Sequence {
	sliceCopy := make([]Serializable, end-start)
	copy(sliceCopy, list.elements[start:end])

	return &List{underylingList: &ValueList{elements: sliceCopy}}
}

func (list *ValueList) Len() int {
	return len(list.elements)
}

func (list *ValueList) At(ctx *Context, i int) Value {
	return list.elements[i]
}

func (list *ValueList) append(ctx *Context, values ...Serializable) {
	list.elements = append(list.elements, values...)
}

func (l *ValueList) insertElement(ctx *Context, v Value, i Int) {
	length := Int(l.Len())
	if i < 0 || i > length {
		panic(ErrInsertionIndexOutOfRange)
	}
	if i == length {
		l.elements = append(l.elements, v.(Serializable))
	} else {
		l.elements = append(l.elements, nil)
		copy(l.elements[i+1:], l.elements[i:])
		l.elements[i] = v.(Serializable)
	}
}

func (l *ValueList) removePosition(ctx *Context, i Int) {
	if int(i) <= len(l.elements)-1 {
		copy(l.elements[i:], l.elements[i+1:])
	}
	l.elements = l.elements[:len(l.elements)-1]
}

func (l *ValueList) removePositionRange(ctx *Context, r IntRange) {
	end := int(r.InclusiveEnd())
	start := int(r.Start)

	if end <= len(l.elements)-1 {
		copy(l.elements[start:], l.elements[end+1:])
	}
	l.elements = l.elements[:len(l.elements)-r.Len()]
}

func (l *ValueList) insertSequence(ctx *Context, seq Sequence, i Int) {
	seqLen := seq.Len()
	if seqLen == 0 {
		return
	}

	if cap(l.elements)-len(l.elements) < seqLen {
		newSlice := make([]Serializable, len(l.elements)+seqLen)
		copy(newSlice, l.elements)
		l.elements = newSlice
	} else {
		l.elements = l.elements[:len(l.elements)+seqLen]
	}

	copy(l.elements[int(i)+seqLen:], l.elements[i:])

	for ind := 0; ind < seqLen; ind++ {
		l.elements[int(i)+ind] = seq.At(ctx, ind).(Serializable)
	}
}

func (l *ValueList) appendSequence(ctx *Context, seq Sequence) {
	l.insertSequence(ctx, seq, Int(l.Len()))
}

// IntList implements underylingList
type IntList struct {
	elements     []Int
	constraintId ConstraintId
}

func NewWrappedIntList(elements ...Int) *List {
	return &List{underylingList: newIntList(elements...)}
}

func NewWrappedIntListFrom(elements []Int) *List {
	return &List{underylingList: &IntList{elements: elements}}
}

func newIntList(elements ...Int) *IntList {
	return &IntList{elements: elements}
}

func (list *IntList) ContainsSimple(ctx *Context, v Serializable) bool {
	if !IsSimpleInoxVal(v) {
		panic("only simple values are expected")
	}

	integer, ok := v.(Int)
	if !ok {
		return false
	}

	for _, n := range list.elements {
		if n == integer {
			return true
		}
	}
	return false
}

func (list *IntList) set(ctx *Context, i int, v Value) {
	list.elements[i] = v.(Int)
}

func (list *IntList) SetSlice(ctx *Context, start, end int, seq Sequence) {
	if seq.Len() != end-start {
		panic(errors.New(FormatIndexableShouldHaveLen(end - start)))
	}

	for i := start; i < end; i++ {
		list.elements[i] = seq.At(ctx, i-start).(Int)
	}
}

func (list *IntList) slice(start, end int) Sequence {
	sliceCopy := make([]Int, end-start)
	copy(sliceCopy, list.elements[start:end])

	return &List{underylingList: &IntList{elements: sliceCopy}}
}

func (list *IntList) Len() int {
	return len(list.elements)
}

func (list *IntList) At(ctx *Context, i int) Value {
	return list.elements[i]
}

func (list *IntList) append(ctx *Context, values ...Serializable) {
	for _, val := range values {
		list.elements = append(list.elements, val.(Int))
	}
}

func (l *IntList) insertElement(ctx *Context, v Value, i Int) {
	length := Int(l.Len())
	if i < 0 || i > length {
		panic(ErrInsertionIndexOutOfRange)
	}
	if i == length {
		l.elements = append(l.elements, v.(Int))
	} else {
		l.elements = append(l.elements, 0)
		copy(l.elements[i+1:], l.elements[i:])
		l.elements[i] = v.(Int)
	}
}

func (l *IntList) removePosition(ctx *Context, i Int) {
	if int(i) <= len(l.elements)-1 {
		copy(l.elements[i:], l.elements[i+1:])
	}
	l.elements = l.elements[:len(l.elements)-1]
}

func (l *IntList) removePositionRange(ctx *Context, r IntRange) {
	end := int(r.InclusiveEnd())
	start := int(r.Start)

	if end <= len(l.elements)-1 {
		copy(l.elements[start:], l.elements[end+1:])
	}
	l.elements = l.elements[:len(l.elements)-r.Len()]
}

func (l *IntList) insertSequence(ctx *Context, seq Sequence, i Int) {
	seqLen := seq.Len()
	if seqLen == 0 {
		return
	}

	if cap(l.elements)-len(l.elements) < seqLen {
		newSlice := make([]Int, len(l.elements)+seqLen)
		copy(newSlice, l.elements)
		l.elements = newSlice
	} else {
		l.elements = l.elements[:len(l.elements)+seqLen]
	}

	copy(l.elements[int(i)+seqLen:], l.elements[i:])

	for ind := 0; ind < seqLen; ind++ {
		l.elements[int(i)+ind] = seq.At(ctx, ind).(Int)
	}
}

func (l *IntList) appendSequence(ctx *Context, seq Sequence) {
	l.insertSequence(ctx, seq, Int(l.Len()))
}

// StringList implements underylingList
type StringList struct {
	elements     []StringLike
	constraintId ConstraintId
}

func NewWrappedStringList(elements ...StringLike) *List {
	return &List{underylingList: newStringList(elements...)}
}

func NewWrappedStringListFrom(elements []StringLike) *List {
	return &List{underylingList: &StringList{elements: elements}}
}

func newStringList(elements ...StringLike) *StringList {
	return &StringList{elements: elements}
}

func (list *StringList) ContainsSimple(ctx *Context, v Serializable) bool {
	if !IsSimpleInoxVal(v) {
		panic("only simple values are expected")
	}

	str, ok := v.(StringLike)
	if !ok {
		return false
	}

	for _, n := range list.elements {
		if n.GetOrBuildString() == str.GetOrBuildString() {
			return true
		}
	}
	return false
}

func (list *StringList) set(ctx *Context, i int, v Value) {
	list.elements[i] = v.(StringLike)
}

func (list *StringList) SetSlice(ctx *Context, start, end int, seq Sequence) {
	if seq.Len() != end-start {
		panic(errors.New(FormatIndexableShouldHaveLen(end - start)))
	}

	for i := start; i < end; i++ {
		list.elements[i] = seq.At(ctx, i-start).(Str)
	}
}

func (list *StringList) slice(start, end int) Sequence {
	sliceCopy := make([]StringLike, end-start)
	copy(sliceCopy, list.elements[start:end])

	return &List{underylingList: &StringList{elements: sliceCopy}}
}

func (list *StringList) Len() int {
	return len(list.elements)
}

func (list *StringList) At(ctx *Context, i int) Value {
	return list.elements[i]
}

func (list *StringList) append(ctx *Context, values ...Serializable) {
	for _, val := range values {
		list.elements = append(list.elements, val.(StringLike))
	}
}

func (l *StringList) insertElement(ctx *Context, v Value, i Int) {
	length := Int(l.Len())
	if i < 0 || i > length {
		panic(ErrInsertionIndexOutOfRange)
	}
	if i == length {
		l.elements = append(l.elements, v.(StringLike))
	} else {
		l.elements = append(l.elements, nil)
		copy(l.elements[i+1:], l.elements[i:])
		l.elements[i] = v.(StringLike)
	}
}

func (l *StringList) removePosition(ctx *Context, i Int) {
	if int(i) <= len(l.elements)-1 {
		copy(l.elements[i:], l.elements[i+1:])
	}
	l.elements = l.elements[:len(l.elements)-1]
}

func (l *StringList) removePositionRange(ctx *Context, r IntRange) {
	end := int(r.InclusiveEnd())
	start := int(r.Start)

	if end <= len(l.elements)-1 {
		copy(l.elements[start:], l.elements[end+1:])
	}
	l.elements = l.elements[:len(l.elements)-r.Len()]
}

func (l *StringList) insertSequence(ctx *Context, seq Sequence, i Int) {
	seqLen := seq.Len()
	if seqLen == 0 {
		return
	}

	if cap(l.elements)-len(l.elements) < seqLen {
		newSlice := make([]StringLike, len(l.elements)+seqLen)
		copy(newSlice, l.elements)
		l.elements = newSlice
	} else {
		l.elements = l.elements[:len(l.elements)+seqLen]
	}

	copy(l.elements[int(i)+seqLen:], l.elements[i:])

	for ind := 0; ind < seqLen; ind++ {
		l.elements[int(i)+ind] = seq.At(ctx, ind).(StringLike)
	}
}

func (l *StringList) appendSequence(ctx *Context, seq Sequence) {
	l.insertSequence(ctx, seq, Int(l.Len()))
}

// BoolList implements underylingList
type BoolList struct {
	elements     *bitset.BitSet
	constraintId ConstraintId
}

func NewWrappedBoolList(elements ...Bool) *List {
	return &List{underylingList: newBoolList(elements...)}
}

func newBoolList(elements ...Bool) *BoolList {
	bitset := bitset.New(uint(len(elements)))
	for i, boolean := range elements {
		if boolean {
			bitset.Set(uint(i))
		}
	}
	return &BoolList{elements: bitset}
}

func (list *BoolList) ContainsSimple(ctx *Context, v Serializable) bool {
	if !IsSimpleInoxVal(v) {
		panic("only booleans are expected")
	}

	boolean, ok := v.(Bool)
	if !ok {
		return false
	}

	if boolean {
		_, ok := list.elements.NextSet(0)
		return ok
	}

	_, ok = list.elements.NextClear(0)
	return ok
}

func (list *BoolList) set(ctx *Context, i int, v Value) {
	boolean := v.(Bool)
	list.elements.SetTo(uint(i), bool(boolean))
}

func (list *BoolList) SetSlice(ctx *Context, start, end int, seq Sequence) {
	if seq.Len() != end-start {
		panic(errors.New(FormatIndexableShouldHaveLen(end - start)))
	}

	for i := start; i < end; i++ {
		list.elements.SetTo(uint(i), bool(seq.At(ctx, i-start).(Bool)))
	}
}

func (list *BoolList) slice(start, end int) Sequence {
	bitSet := bitset.New(uint(end - start))
	newIndex := uint(0)

	for i := uint(start); i < uint(end); i, newIndex = i+1, newIndex+1 {
		bitSet.SetTo(newIndex, list.elements.Test(i))
	}

	return &BoolList{elements: bitSet}
}

func (list *BoolList) Len() int {
	return int(list.elements.Len())
}

func (list *BoolList) BoolAt(i int) bool {
	return list.elements.Test(uint(i))
}

func (list *BoolList) At(ctx *Context, i int) Value {
	return Bool(list.BoolAt(i))
}

func (list *BoolList) append(ctx *Context, values ...Serializable) {
	newLength := list.Len() + len(values)
	newBitSet := bitset.New(uint(newLength))
	copied := list.elements.Copy(newBitSet)
	if copied != uint(list.Len()) {
		panic(ErrUnreachable)
	}
	list.elements = newBitSet
}

func (l *BoolList) insertElement(ctx *Context, v Value, i Int) {
	l.elements.InsertAt(uint(i))
	l.set(ctx, int(i), v)
}

func (l *BoolList) removePosition(ctx *Context, i Int) {
	l.elements.DeleteAt(uint(i))
}

func (l *BoolList) removePositionRange(ctx *Context, r IntRange) {
	index := int(r.Start)
	for i := 0; i < r.Len(); i++ {
		l.elements.DeleteAt(uint(index))
	}
}

func (l *BoolList) insertSequence(ctx *Context, seq Sequence, i Int) {
	seqLen := seq.Len()
	for ind := seqLen - 1; ind >= 0; ind-- {
		l.insertElement(ctx, seq.At(ctx, ind).(Serializable), i)
	}
}

func (l *BoolList) appendSequence(ctx *Context, seq Sequence) {
	l.insertSequence(ctx, seq, Int(l.Len()))
}

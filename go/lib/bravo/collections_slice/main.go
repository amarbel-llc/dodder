package collections_slice

import (
	"slices"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type Slice[ELEMENT any] []ELEMENT

var _ interfaces.Collection[string] = Slice[string]{}

func Make[ELEMENT any](elements ...ELEMENT) Slice[ELEMENT] {
	return Slice[ELEMENT](elements)
}

func MakeFromSeq[ELEMENT any](count int, seq interfaces.Seq[ELEMENT]) Slice[ELEMENT] {
	slice := make(Slice[ELEMENT], count)

	for element := range seq {
		slice.Append(element)
	}

	return slice
}

func MakeFromSlice[ELEMENT any](elements ...ELEMENT) Slice[ELEMENT] {
	slice := make(Slice[ELEMENT], len(elements))

	copy(slice, elements)

	return slice
}

func MakeWithLen[ELEMENT any](length int) Slice[ELEMENT] {
	slice := make(Slice[ELEMENT], length)
	return slice
}

func MakeWithCap[ELEMENT any](capacity int) Slice[ELEMENT] {
	slice := make(Slice[ELEMENT], 0, capacity)
	return slice
}

func Collect[ELEMENT any](seq interfaces.Seq[ELEMENT]) Slice[ELEMENT] {
	return Slice[ELEMENT](slices.Collect(seq))
}

func (slice Slice[ELEMENT]) Len() int {
	return len(slice)
}

func (slice Slice[ELEMENT]) IsEmpty() bool {
	return slice.Len() == 0
}

func (slice Slice[ELEMENT]) Any() (element ELEMENT) {
	if slice.Len() > 0 {
		element = slice[0]
	}

	return element
}

func (slice Slice[ELEMENT]) First() ELEMENT {
	return slice[0]
}

func (slice Slice[ELEMENT]) Last() ELEMENT {
	return slice[slice.Len()-1]
}

func (slice Slice[ELEMENT]) SetLast(element ELEMENT) {
	slice[slice.Len()-1] = element
}

func (slice Slice[ELEMENT]) All() interfaces.Seq[ELEMENT] {
	return func(yield func(ELEMENT) bool) {
		for _, element := range slice {
			if !yield(element) {
				break
			}
		}
	}
}

func (slice *Slice[ELEMENT]) Grow(n int) {
	*slice = slices.Grow(*slice, n)
}

func (slice *Slice[ELEMENT]) DropFirst() {
	if slice.Len() > 0 {
		*slice = (*slice)[1:]
	}
}

func (slice *Slice[ELEMENT]) DropLast() {
	*slice = (*slice)[:slice.Len()-1]
}

func (slice *Slice[ELEMENT]) Delete(leftInclusive, rightExclusive int) {
	*slice = slices.Delete[[]ELEMENT](*slice, leftInclusive, rightExclusive)
}

func (slice *Slice[ELEMENT]) At(index int) ELEMENT {
	return (*slice)[index]
}

func (slice *Slice[ELEMENT]) Append(elements ...ELEMENT) {
	*slice = append(*slice, elements...)
}

func (slice *Slice[ELEMENT]) Merge(otherSeq Slice[ELEMENT]) {
	*slice = append(*slice, otherSeq...)
}

func (slice *Slice[ELEMENT]) Reset() {
	*slice = (*slice)[:0]
}

func (slice *Slice[ELEMENT]) ResetWith(other Slice[ELEMENT]) {
	slice.ResetWithCollection(other)
}

func (slice *Slice[ELEMENT]) ResetWithCollection(
	collection interfaces.Collection[ELEMENT],
) {
	slice.Reset()
	slice.Grow(collection.Len())

	for element := range collection.All() {
		slice.Append(element)
	}
}

func (slice *Slice[ELEMENT]) ResetWithSeq(other interfaces.Seq[ELEMENT]) {
	slice.Reset()

	for element := range other {
		slice.Append(element)
	}
}

func (slice *Slice[ELEMENT]) Clone() Slice[ELEMENT] {
	return MakeFromSlice(*slice...)
}

func (slice Slice[ELEMENT]) Swap(left, right int) {
	slice[right], slice[left] = slice[left], slice[right]
}

func (slice *Slice[ELEMENT]) Insert(index int, values ...ELEMENT) {
	*slice = slices.Insert(*slice, index, values...)
}

func (slice Slice[ELEMENT]) Shift(amount int) Slice[ELEMENT] {
	return slice[amount:]
}

func (slice *Slice[ELEMENT]) ShiftInPlace(amount int) {
	*slice = slice.Shift(amount)
}

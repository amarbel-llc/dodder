package objects

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/charlie/expansion"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type ContainedObjects collections_slice.Slice[containedObject]

func (contents ContainedObjects) GetSlice() collections_slice.Slice[containedObject] {
	return collections_slice.Slice[containedObject](contents)
}

func (contents *ContainedObjects) GetSliceMutable() *collections_slice.Slice[containedObject] {
	return (*collections_slice.Slice[containedObject])(contents)
}

func (contents ContainedObjects) Len() int {
	return contents.GetSlice().Len()
}

func (contents ContainedObjects) All() interfaces.Seq[SeqId] {
	return func(yield func(SeqId) bool) {
		for id := range contents.GetSlice().All() {
			if !yield(id.GetKey()) {
				return
			}
		}
	}
}

func (contents ContainedObjects) ContainsKey(key string) bool {
	_, ok := cmp.BinarySearchFuncIndex(
		contents.GetSlice(),
		key,
		func(left containedObject, right string) cmp.Result {
			return cmp.CompareUTF8(
				left.GetKey().Seq.GetComparable(),
				cmp.ComparableString(right),
				false,
			)
		},
	)

	return ok
}

func (contents ContainedObjects) getLock(key string) (IdLock, bool) {
	for id := range contents.GetSlice().All() {
		if id.GetKey().String() == key {
			return id.Lock, true
		}
	}

	return nil, false
}

func (contents ContainedObjects) getLockMutable(key string) (IdLockMutable, bool) {
	for index := range contents {
		id := &contents[index]

		if id.GetKey().String() == key {
			return &id.Lock, true
		}
	}

	return nil, false
}

func (contents ContainedObjects) Get(key string) (SeqId, bool) {
	id, ok := contents.get(key, false)
	return id.GetKey(), ok
}

func (contents ContainedObjects) GetPartial(key string) (SeqId, bool) {
	id, ok := contents.get(key, true)
	return id.GetKey(), ok
}

func (contents ContainedObjects) get(key string, partial bool) (containedObject, bool) {
	element, ok := cmp.BinarySearchFuncElement(
		contents,
		key,
		func(left containedObject, right string) cmp.Result {
			return cmp.CompareUTF8(
				left.GetKey().Seq.GetComparable(),
				cmp.ComparableString(right),
				partial,
			)
		},
	)

	return element, ok
}

func (contents ContainedObjects) Key(id SeqId) string {
	return id.String()
}

func (contents *ContainedObjects) Add(id SeqId) error {
	if _, alreadyExists := contents.Get(id.String()); alreadyExists {
		return nil
	}

	contents.GetSliceMutable().Append(containedObject{
		Lock: markl.MakeLockWith(id, nil),
	})

	contents.GetSliceMutable().SortWithComparer(containedObjectCompareKey)

	return nil
}

func (contents *ContainedObjects) DelKey(key string) error {
	var found bool
	var index int
	var id containedObject

	for index, id = range contents.GetSlice() {
		if id.GetKey().String() == key {
			found = true
			break
		}
	}

	if found {
		contents.GetSliceMutable().Delete(index, index+1)
	}

	return nil
}

func (contents *ContainedObjects) Reset() {
	contents.GetSliceMutable().Reset()
}

// TODO add optimized non-sorted path for binary decoding
func (contents *ContainedObjects) addNormalizedTag(tag Tag) {
	seq := expansion.ExpandOneIntoIds[SeqId](
		tag.String(),
		expansion.ExpanderRight,
	)

	for id, err := range seq {
		errors.PanicIfError(err)
		errors.PanicIfError(contents.Add(id))
	}

	sorted := quiter.SortedValuesBy(
		contents.GetSlice(),
		containedObjectCompareKey,
	)

	var lastId *containedObject

	for index := range sorted {
		id := &sorted[index]

		if index == 0 {
			// no need to do anything, this is the first
			lastId = id
			continue
		}

		tagString := id.Lock.GetKey().String()
		lastTagString := lastId.Lock.GetKey().String()

		switch {
		case strings.HasPrefix(lastTagString, tagString):
			continue

			// replace the shorter value with the longer value that contains the
			// shorter value
		case strings.HasPrefix(tagString, lastTagString):
			if lastId.Lock.Value.IsEmpty() {
				contents.DelKey(lastTagString)
			}

		default:
			// keep the tag
		}

		lastId = id
	}
}

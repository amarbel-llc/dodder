package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	contentsTagSet struct {
		*ContainedObjects
	}
)

var (
	_ TagSet        = &contentsTagSet{}
	_ TagSetMutable = &contentsTagSet{}
)

func (contentsTagSet contentsTagSet) Len() int {
	return contentsTagSet.GetSlice().Len()
}

func (contentsTagSet contentsTagSet) All() interfaces.Seq[TagStruct] {
	return func(yield func(TagStruct) bool) {
		for id := range contentsTagSet.GetSlice().All() {
			var tag TagStruct

			errors.PanicIfError(tag.Set(id.GetKey().String()))

			if !yield(tag) {
				return
			}
		}
	}
}

// TODO switch to binary search
func (contentsTagSet contentsTagSet) ContainsKey(key string) bool {
	for id := range contentsTagSet.All() {
		if id.String() == key {
			return true
		}
	}

	return false
}

// TODO switch to binary search
func (contentsTagSet contentsTagSet) Get(key string) (TagStruct, bool) {
	for tag := range contentsTagSet.ContainedObjects.GetSlice().All() {
		if tag.GetKey().String() == key {
			return ids.MustTag(tag.GetKey().String()), true
		}
	}

	return TagStruct{}, false
}

func (contentsTagSet contentsTagSet) Key(tag TagStruct) string {
	return tag.String()
}

// TODO sort
func (contentsTagSet *contentsTagSet) Add(tag TagStruct) error {
	var tagId SeqId

	if err := tagId.Set(tag.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return contentsTagSet.ContainedObjects.Add(tagId)
}

func (contentsTagSet *contentsTagSet) DelKey(key string) error {
	return contentsTagSet.ContainedObjects.DelKey(key)
}

func (contentsTagSet *contentsTagSet) Reset() {
	contentsTagSet.ContainedObjects.Reset()
}

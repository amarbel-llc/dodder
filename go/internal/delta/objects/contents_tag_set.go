package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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
	return contentsTagSet.ContainedObjects.TagLen()
}

func (contentsTagSet contentsTagSet) All() interfaces.Seq[TagStruct] {
	return func(yield func(TagStruct) bool) {
		for id := range contentsTagSet.ContainedObjects.AllTags() {
			var tag TagStruct

			errors.PanicIfError(tag.Set(id.String()))

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
	for tag := range contentsTagSet.ContainedObjects.AllTags() {
		if tag.String() == key {
			return ids.MustTag(tag.String()), true
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
	contentsTagSet.ContainedObjects.ResetTags()
}

package objects

import (
	_ "encoding/gob"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/echo/descriptions"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

// TODO transform into two views that satisfy the Metadata/MetadataMutable
// interfaces:
// - struct like the current one
// - index bytes, like the representation used by stream_index
type metadata struct {
	// all the fiels need to be public for gob's stupid illusions, but should be
	// private once moving away from the gob entirely

	Description descriptions.Description
	Tags        ContainedObjects
	Type        markl.Lock[Type, TypeMutable]

	DigBlob   markl.Id
	digSelf   markl.Id
	pubRepo   markl.Id
	sigMother markl.Id
	sigRepo   markl.Id

	Tai ids.Tai

	Index index
}

var (
	_ Metadata        = &metadata{}
	_ MetadataMutable = &metadata{}
	_ Getter          = &metadata{}
	_ GetterMutable   = &metadata{}
)

func Make() *metadata {
	metadata := &metadata{}
	Resetter.Reset(metadata)
	return metadata
}

func (metadata *metadata) GetMetadata() Metadata {
	return metadata
}

func (metadata *metadata) GetMetadataMutable() MetadataMutable {
	return metadata
}

func (metadata *metadata) GetIndex() Index {
	return &metadata.Index
}

func (metadata *metadata) GetIndexMutable() IndexMutable {
	return &metadata.Index
}

func (metadata *metadata) GetDescription() descriptions.Description {
	return metadata.Description
}

func (metadata *metadata) GetDescriptionMutable() *descriptions.Description {
	return &metadata.Description
}

func (metadata *metadata) GetTai() ids.Tai {
	return metadata.Tai
}

func (metadata *metadata) GetTaiMutable() *ids.Tai {
	return &metadata.Tai
}

func (metadata *metadata) UserInputIsEmpty() bool {
	if !metadata.Description.IsEmpty() {
		return false
	}

	if metadata.Tags.Len() > 0 {
		return false
	}

	if !ids.IsEmpty(metadata.GetType()) {
		return false
	}

	return true
}

func (metadata *metadata) IsEmpty() bool {
	if !metadata.DigBlob.IsNull() {
		return false
	}

	if !metadata.UserInputIsEmpty() {
		return false
	}

	if !metadata.Tai.IsZero() {
		return false
	}

	return true
}

// TODO fix issue with GetTags being nil sometimes
func (metadata *metadata) GetTags() TagSet {
	return contentsTagSet{ContainedObjects: &metadata.Tags}
}

func (metadata *metadata) GetTagsMutable() TagSetMutable {
	return &contentsTagSet{ContainedObjects: &metadata.Tags}
}

func (metadata *metadata) AllTags() interfaces.Seq[Tag] {
	return func(yield func(Tag) bool) {
		for tag := range metadata.Tags.All() {
			if !yield(tag) {
				return
			}
		}
	}
}

func (metadata *metadata) ResetTags() {
	metadata.Tags.Reset()
	metadata.Index.TagPaths.Reset()
}

func (metadata *metadata) AddTagString(tagString string) (err error) {
	if tagString == "" {
		return err
	}

	var tag ids.TagStruct

	if err = tag.Set(tagString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = metadata.AddTagPtr(tag); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (metadata *metadata) AddTag(tag Tag) (err error) {
	return metadata.AddTagPtr(tag)
}

func (metadata *metadata) AddTagPtr(tag Tag) (err error) {
	if tag.IsEmpty() {
		return err
	}

	metadata.Tags.addNormalizedTag(tag)
	cs, _ := catgut.MakeFromString(tag.String())
	metadata.Index.TagPaths.AddTag(cs)

	return err
}

func (metadata *metadata) AddTagPtrFast(tag Tag) (err error) {
	ids.TagSetMutableAdd(metadata.GetTagsMutable(), tag)

	tagBytestring, _ := catgut.MakeFromString(tag.String())

	if err = metadata.Index.TagPaths.AddTag(tagBytestring); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (metadata *metadata) SetTagsFast(tags TagSet) {
	metadata.Tags.Reset()

	if tags == nil {
		return
	}

	if tags.Len() == 1 && quiter_set.Any(tags).String() == "" {
		panic("empty tag set")
	}

	for tag := range tags.All() {
		errors.PanicIfError(metadata.AddTagPtrFast(tag))
	}
}

func (metadata *metadata) GetType() Type {
	return metadata.Type.Key
	// id, ok := metadata.Contents.GetPartial("!")

	// if !ok {
	// 	panic("missing type")
	// }

	// return ids.MustType(id.String())
}

func (metadata *metadata) GetTypeMutable() TypeMutable {
	return &metadata.Type.Key
}

func (metadata *metadata) GetTypeLock() TypeLock {
	return &metadata.Type
}

func (metadata *metadata) GetTypeLockMutable() TypeLockMutable {
	return &metadata.Type
}

func (metadata *metadata) GetTagLock(tag Tag) TagLock {
	lock, _ := metadata.Tags.getLock(tag.String())
	return lock
}

func (metadata *metadata) GetTagLockMutable(tag Tag) TagLockMutable {
	lock, _ := metadata.Tags.getLockMutable(tag.String())
	return lock
}

func (metadata *metadata) Subtract(otherMetadata Metadata) {
	if metadata.GetType().String() == otherMetadata.GetType().String() {
		metadata.GetTypeMutable().Reset()
	}

	for tag := range otherMetadata.AllTags() {
		metadata.Tags.DelKey(tag.String())
	}
}

func (metadata *metadata) GenerateExpandedTags() {
}

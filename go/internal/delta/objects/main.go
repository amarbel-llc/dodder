package objects

import (
	_ "encoding/gob"

	"code.linenisgreat.com/dodder/go/internal/bravo/descriptions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

// TODO transform into two views that satisfy the Metadata/MetadataMutable
// interfaces:
// - struct like the current one
// - index bytes, like the representation used by stream_index
type metadata struct {
	// all the fiels need to be public for gob's stupid illusions, but should be
	// private once moving away from the gob entirely

	Description descriptions.Description
	Contents    ContainedObjects
	BlobRefs    BlobReferences
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

	if metadata.Contents.TagLen() > 0 {
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
	return contentsTagSet{ContainedObjects: &metadata.Contents}
}

func (metadata *metadata) GetTagsMutable() TagSetMutable {
	return &contentsTagSet{ContainedObjects: &metadata.Contents}
}

func (metadata *metadata) AllTags() interfaces.Seq[Tag] {
	return func(yield func(Tag) bool) {
		for tag := range metadata.Contents.AllTags() {
			if !yield(tag) {
				return
			}
		}
	}
}

func (metadata *metadata) ResetTags() {
	metadata.Contents.ResetTags()
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

	metadata.Contents.addNormalizedTag(tag)
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
	metadata.Contents.ResetTags()

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
	lock, _ := metadata.Contents.getLock(tag.String())
	return lock
}

func (metadata *metadata) GetTagLockMutable(tag Tag) TagLockMutable {
	lock, _ := metadata.Contents.getLockMutable(tag.String())
	return lock
}

func (metadata *metadata) AllReferencedObjects() interfaces.Seq[SeqId] {
	return func(yield func(SeqId) bool) {
		for ref := range metadata.Contents.AllReferences() {
			if !yield(ref) {
				return
			}
		}
	}
}

func (metadata *metadata) GetReferencedObjectLock(ref SeqId) IdLock {
	lock, _ := metadata.Contents.getLock(ref.String())
	return lock
}

func (metadata *metadata) GetReferencedObjectLockMutable(ref SeqId) IdLockMutable {
	lock, _ := metadata.Contents.getLockMutable(ref.String())
	return lock
}

func (metadata *metadata) AddReference(ref SeqId) error {
	return metadata.Contents.AddReference(ref)
}

func (metadata *metadata) SetReferenceAlias(ref SeqId, alias string) error {
	for index := range metadata.Contents {
		entry := &metadata.Contents[index]

		if !entry.ContainedObjectType.IsReference() {
			continue
		}

		if entry.GetKey().String() == ref.String() {
			return entry.Alias.Set(alias)
		}
	}

	return errors.Errorf("reference not found: %s", ref)
}

func (metadata *metadata) GetReferenceAlias(ref SeqId) string {
	for index := range metadata.Contents {
		entry := &metadata.Contents[index]

		if !entry.ContainedObjectType.IsReference() {
			continue
		}

		if entry.GetKey().String() == ref.String() {
			return entry.Alias.String()
		}
	}

	return ""
}

func (metadata *metadata) AllBlobReferences() interfaces.Seq[markl.Id] {
	return metadata.BlobRefs.All()
}

func (metadata *metadata) AddBlobReference(id markl.Id) {
	metadata.BlobRefs.Add(id)
}

func (metadata *metadata) SetBlobReferenceAlias(id markl.Id, alias string) error {
	return metadata.BlobRefs.SetAlias(id, alias)
}

func (metadata *metadata) GetBlobReferenceAlias(id markl.Id) string {
	return metadata.BlobRefs.GetAlias(id)
}

func (metadata *metadata) ResetBlobReferences() {
	metadata.BlobRefs.Reset()
}

func (metadata *metadata) Subtract(otherMetadata Metadata) {
	if metadata.GetType().String() == otherMetadata.GetType().String() {
		metadata.GetTypeMutable().Reset()
	}

	for tag := range otherMetadata.AllTags() {
		metadata.Contents.DelKey(tag.String())
	}
}

func (metadata *metadata) GenerateExpandedTags() {
}

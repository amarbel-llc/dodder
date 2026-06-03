package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/descriptions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// TODO transform into two views that satisfy the Metadata/MetadataMutable
// interfaces:
// - struct like the current one
// - index bytes, like the representation used by stream_index
type metadata struct {
	description descriptions.Description
	contents    ContainedObjects
	blobRefs    BlobReferences
	typ         markl.Lock[Type, TypeMutable]

	digBlob   markl.Id
	digSelf   markl.Id
	pubRepo   markl.Id
	sigMother markl.Id
	sigRepo   markl.Id

	tai ids.Tai

	idx index
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
	return &metadata.idx
}

func (metadata *metadata) GetIndexMutable() IndexMutable {
	return &metadata.idx
}

func (metadata *metadata) GetDescription() descriptions.Description {
	return metadata.description
}

func (metadata *metadata) GetDescriptionMutable() *descriptions.Description {
	return &metadata.description
}

func (metadata *metadata) GetTai() ids.Tai {
	return metadata.tai
}

func (metadata *metadata) GetTaiMutable() *ids.Tai {
	return &metadata.tai
}

func (metadata *metadata) UserInputIsEmpty() bool {
	if !metadata.description.IsEmpty() {
		return false
	}

	if metadata.contents.TagLen() > 0 {
		return false
	}

	if !ids.IsEmpty(metadata.GetType()) {
		return false
	}

	return true
}

func (metadata *metadata) IsEmpty() bool {
	if !metadata.digBlob.IsNull() {
		return false
	}

	if !metadata.UserInputIsEmpty() {
		return false
	}

	if !metadata.tai.IsZero() {
		return false
	}

	return true
}

// TODO fix issue with GetTags being nil sometimes
func (metadata *metadata) GetTags() TagSet {
	return contentsTagSet{ContainedObjects: &metadata.contents}
}

func (metadata *metadata) GetTagsMutable() TagSetMutable {
	return &contentsTagSet{ContainedObjects: &metadata.contents}
}

func (metadata *metadata) AllTags() interfaces.Seq[Tag] {
	return func(yield func(Tag) bool) {
		for tag := range metadata.contents.AllTags() {
			if !yield(tag) {
				return
			}
		}
	}
}

func (metadata *metadata) ResetTags() {
	metadata.contents.ResetTags()
	metadata.idx.TagPaths.Reset()
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

	metadata.contents.addNormalizedTag(tag)
	cs, _ := catgut.MakeFromString(tag.String()) //repool:owned
	metadata.idx.TagPaths.AddTag(cs)

	return err
}

func (metadata *metadata) AddTagPtrFast(tag Tag) (err error) {
	ids.TagSetMutableAdd(metadata.GetTagsMutable(), tag)

	tagBytestring, _ := catgut.MakeFromString(tag.String()) //repool:owned

	if err = metadata.idx.TagPaths.AddTag(tagBytestring); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (metadata *metadata) SetTagsFast(tags TagSet) {
	metadata.contents.ResetTags()

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
	return metadata.typ.Key
	// id, ok := metadata.Contents.GetPartial("!")

	// if !ok {
	// 	panic("missing type")
	// }

	// return ids.MustType(id.String())
}

func (metadata *metadata) GetTypeMutable() TypeMutable {
	return &metadata.typ.Key
}

func (metadata *metadata) GetTypeLock() TypeLock {
	return &metadata.typ
}

func (metadata *metadata) GetTypeLockMutable() TypeLockMutable {
	return &metadata.typ
}

func (metadata *metadata) GetTagLock(tag Tag) TagLock {
	lock, _ := metadata.contents.getLock(tag.String())
	return lock
}

func (metadata *metadata) GetTagLockMutable(tag Tag) TagLockMutable {
	lock, _ := metadata.contents.getLockMutable(tag.String())
	return lock
}

func (metadata *metadata) AllReferencedObjects() interfaces.Seq[SeqId] {
	return func(yield func(SeqId) bool) {
		for ref := range metadata.contents.AllReferences() {
			if !yield(ref) {
				return
			}
		}
	}
}

func (metadata *metadata) GetReferencedObjectLock(ref SeqId) IdLock {
	lock, _ := metadata.contents.getLock(ref.String())
	return lock
}

func (metadata *metadata) GetReferencedObjectLockMutable(ref SeqId) IdLockMutable {
	lock, _ := metadata.contents.getLockMutable(ref.String())
	return lock
}

func (metadata *metadata) AddReference(ref SeqId) error {
	return metadata.contents.AddReference(ref)
}

func (metadata *metadata) SetReferenceAlias(ref SeqId, alias string) error {
	for index := range metadata.contents {
		entry := &metadata.contents[index]

		if !entry.ContainedObjectType.IsReference() {
			continue
		}

		if entry.GetKey().String() == ref.String() {
			entry.Alias = alias
			return nil
		}
	}

	return errors.Errorf("reference not found: %s", ref)
}

func (metadata *metadata) GetReferenceAlias(ref SeqId) string {
	for index := range metadata.contents {
		entry := &metadata.contents[index]

		if !entry.ContainedObjectType.IsReference() {
			continue
		}

		if entry.GetKey().String() == ref.String() {
			return entry.Alias
		}
	}

	return ""
}

func (metadata *metadata) AllBlobReferences() interfaces.Seq[markl.Id] {
	return metadata.blobRefs.All()
}

func (metadata *metadata) AddBlobReference(
	id markl.Id,
	typeLock markl.Lock[ids.SeqId, *ids.SeqId],
) {
	metadata.blobRefs.Add(id, typeLock)
}

func (metadata *metadata) SetBlobReferenceAlias(id markl.Id, alias string) error {
	return metadata.blobRefs.SetAlias(id, alias)
}

func (metadata *metadata) GetBlobReferenceAlias(id markl.Id) string {
	return metadata.blobRefs.GetAlias(id)
}

func (metadata *metadata) GetBlobReferenceTypeLock(
	id markl.Id,
) markl.Lock[ids.SeqId, *ids.SeqId] {
	return metadata.blobRefs.GetTypeLock(id)
}

func (metadata *metadata) GetBlobReferenceTypeLockMutable(
	id markl.Id,
) *markl.Lock[ids.SeqId, *ids.SeqId] {
	return metadata.blobRefs.GetTypeLockMutable(id)
}

func (metadata *metadata) ResetBlobReferences() {
	metadata.blobRefs.Reset()
}

func (metadata *metadata) Subtract(otherMetadata Metadata) {
	if metadata.GetType().String() == otherMetadata.GetType().String() {
		metadata.GetTypeMutable().Reset()
	}

	for tag := range otherMetadata.AllTags() {
		metadata.contents.DelKey(tag.String())
	}
}

func (metadata *metadata) GenerateExpandedTags() {
}

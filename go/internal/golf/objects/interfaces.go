package objects

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/descriptions"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

type (
	MetadataStruct = metadata

	TagStruct     = ids.TagStruct
	Tag           = ids.Tag
	TagSet        = ids.Set[ids.TagStruct]
	TagSetMutable = ids.SetMutable[ids.TagStruct]

	IdLock        = domain_interfaces.Lock[SeqId, *SeqId]
	IdLockMutable = domain_interfaces.LockMutable[SeqId, *SeqId]

	TypeLock        = domain_interfaces.Lock[Type, TypeMutable]
	TypeLockMutable = domain_interfaces.LockMutable[Type, TypeMutable]
	TagLock         = IdLock
	TagLockMutable  = IdLockMutable

	Type        = ids.SeqId
	TypeMutable = *ids.SeqId

	Metadata interface {
		Getter

		IsEmpty() bool

		GetDescription() descriptions.Description
		GetIndex() Index
		GetTags() TagSet
		AllTags() interfaces.Seq[Tag]
		GetTai() ids.Tai
		GetType() Type
		GetTypeLock() TypeLock

		GetTagLock(Tag) TagLock

		GetBlobDigest() domain_interfaces.MarklId
		GetObjectDigest() domain_interfaces.MarklId
		GetMotherObjectSig() domain_interfaces.MarklId
		GetRepoPubKey() domain_interfaces.MarklId
		GetObjectSig() domain_interfaces.MarklId
	}

	MetadataMutable interface {
		interfaces.CommandComponentWriter
		Metadata
		GetterMutable

		Subtract(Metadata)

		AddTagPtr(Tag) (err error)
		ResetTags()
		AddTagString(tagString string) (err error)
		AddTagPtrFast(tag Tag) (err error)
		GenerateExpandedTags()

		GetIndexMutable() IndexMutable

		GetBlobDigestMutable() domain_interfaces.MarklIdMutable
		GetDescriptionMutable() *descriptions.Description
		GetMotherObjectSigMutable() domain_interfaces.MarklIdMutable
		GetObjectDigestMutable() domain_interfaces.MarklIdMutable
		GetObjectSigMutable() domain_interfaces.MarklIdMutable
		GetRepoPubKeyMutable() domain_interfaces.MarklIdMutable
		GetTaiMutable() *ids.Tai
		GetTypeMutable() TypeMutable
		GetTypeLockMutable() TypeLockMutable
		GetTagLockMutable(Tag) TagLockMutable
	}

	Getter interface {
		GetMetadata() Metadata
	}

	GetterMutable interface {
		GetMetadataMutable() MetadataMutable
	}
)

// Coding
type (
	EncoderContext interface {
		Getter
		// GetObjectId() ids.Id
	}

	DecoderContext interface {
		GetterMutable
		SetBlobDigest(domain_interfaces.MarklId) error
		// GetObjectIdMutable() ids.IdMutable
	}
)

package sku

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	FSItemReadWriter interface {
		ReadFSItemFromExternal(TransactedGetter) (*FSItem, error)
		WriteFSItemToExternal(*FSItem, TransactedGetter) (err error)
	}

	OneReader interface {
		ReadTransactedFromObjectId(
			k1 ids.Id,
		) (sk1 *Transacted, err error)
	}

	BlobSaver interface {
		SaveBlob(ExternalLike) (err error)
	}
	IdAbbrIndexPresenceGeneric[_ any] interface {
		Exists([3]string) error
	}

	IdAbbrIndexGeneric[ID any, ID_PTR interfaces.Ptr[ID]] interface {
		IdAbbrIndexPresenceGeneric[ID]
		ExpandStringString(string) (string, error)
		ExpandString(string) (ID_PTR, error)
		Expand(ID_PTR) (ID_PTR, error)
		Abbreviate(domain_interfaces.Abbreviatable) (string, error)
	}

	IdIndex interface {
		GetSeenIds() map[genres.Genre]interfaces.Collection[string]
		GetZettelIds() IdAbbrIndexGeneric[ids.ZettelId, *ids.ZettelId]
		GetBlobIds() IdAbbrIndexGeneric[markl.Id, *markl.Id]

		AddObjectToIdIndex(*Transacted) error
		GetAbbr() ids.Abbr

		errors.Flusher
	}

	RepoStore interface {
		Commit(*Transacted, CommitOptions) (err error)
		ReadOneInto(domain_interfaces.ObjectId, *Transacted) (err error)
		ReadPrimitiveQuery(
			qg PrimitiveQueryGroup,
			w interfaces.FuncIter[*Transacted],
		) (err error)
	}

	ExternalObjectId       = domain_interfaces.ExternalObjectId
	ExternalObjectIdGetter = domain_interfaces.ExternalObjectIdGetter

	FuncReadOneInto = func(
		k1 domain_interfaces.ObjectId,
		out *Transacted,
	) (err error)

	ExternalStoreUpdateTransacted interface {
		UpdateTransacted(z *Transacted) (err error)
	}

	ExternalStoreReadExternalLikeFromObjectIdLike interface {
		ReadExternalLikeFromObjectIdLike(
			o CommitOptions,
			oid interfaces.Stringer,
			t *Transacted,
		) (e ExternalLike, err error)
	}
)

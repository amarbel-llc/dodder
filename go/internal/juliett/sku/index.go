package sku

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type (
	FuncReadOne = func(
		sh domain_interfaces.MarklId,
		sk *Transacted,
	) (ok bool)

	ObjectProbeIndex interface {
		ReadOneObjectId(domain_interfaces.ObjectId, *Transacted) error
	}

	IndexPrimitives interface {
		ObjectExists(
			objectId *ids.ObjectId,
		) (err error)

		// ReadOneMarklId(
		// 	ctx interfaces.ActiveContext,
		// 	marklId interfaces.MarklId,
		// 	object *Transacted,
		// ) (ok bool)

		ReadOneMarklIdAdded(
			sh domain_interfaces.MarklId,
			sk *Transacted,
		) (ok bool)

		ReadOneMarklId(
			sh domain_interfaces.MarklId,
			sk *Transacted,
		) (ok bool)
	}

	Index interface {
		IndexPrimitives
		ObjectProbeIndex

		ReadOneObjectIdTai(
			k ids.Id,
			t ids.Tai,
		) (sk *Transacted, err error)

		ReadManyObjectId(
			id ids.Id,
		) (skus []*Transacted, err error)

		ReadManyMarklId(
			sh domain_interfaces.MarklId,
		) (skus []*Transacted, err error)
	}

	IndexMutation interface {
		Add(
			object *Transacted,
			options CommitOptions,
		) (err error)
	}

	IndexMutable interface {
		Index
		IndexMutation
	}

	Reindexer interface {
		IndexPrimitives
		IndexMutation
	}
)

func ReadOneObjectId(
	index IndexPrimitives,
	objectId ids.Id,
	object *Transacted,
) (ok bool) {
	return ReadOneObjectIdBespoke(
		objectId,
		object,
		index.ReadOneMarklId,
	)
}

func ReadOneObjectIdBespoke(
	objectId domain_interfaces.ObjectId,
	object *Transacted,
	funcs ...FuncReadOne,
) (ok bool) {
	objectIdString := objectId.String()

	if objectIdString == "" {
		panic("empty object id")
	}

	// TODO don't hardcode hash format
	digest, repool := markl.FormatHashSha256.GetMarklIdForString(
		objectIdString,
	)
	defer repool()

	for _, funk := range funcs {
		if ok = funk(digest, object); ok {
			break
		}
	}

	return ok
}

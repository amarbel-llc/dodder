package sku

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	FuncReadOne = func(
		sh mad_domain_interfaces.MarklId,
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
			sh mad_domain_interfaces.MarklId,
			sk *Transacted,
		) (ok bool)

		ReadOneMarklId(
			sh mad_domain_interfaces.MarklId,
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
			sh mad_domain_interfaces.MarklId,
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

	// StreamIndex is the full behavioral contract of a concrete stream index
	// implementation, allowing the store to remain agnostic about whether the
	// variable-length or fixed-size-row-with-overflow index is in use.
	StreamIndex interface {
		IndexMutable

		Reset() (err error)

		Flush(
			printerHeader interfaces.FuncIter[string],
		) (err error)

		ReadPrimitiveQuery(
			queryGroup PrimitiveQueryGroup,
			funcIter interfaces.FuncIter[*Transacted],
		) (err error)

		SetNeedsFlushHistory(changes []string)

		VerifyObjectProbes(object *Transacted) (err error)

		PrintAllProbes() (err error)

		MakeReindexer(
			ctx interfaces.ActiveContext,
		) (reindexer Reindexer, err error)
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

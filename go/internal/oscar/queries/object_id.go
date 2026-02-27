package queries

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/doddish"
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/hotel/objects"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type ObjectId struct {
	Exact   bool
	Virtual bool
	Debug   bool

	*ids.ObjectId

	marklId markl.Id
}

var _ ObjectId = ObjectId{}

func (objectId *ObjectId) reduce(buildState *buildState) (err error) {
	if err = ids.Expand(
		objectId.GetObjectId(),
		buildState.builder.expanders,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if objectId.GetGenre() == genres.Blob {
		if err = objectId.marklId.Set(objectId.GetObjectId().String()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (objectId *ObjectId) ReadFromSeq(seq doddish.Seq) (err error) {
	ok, left, _ := seq.MatchEnd(doddish.TokenMatcherOp(doddish.OpExact))

	if ok {
		objectId.Exact = true
		seq = left
	}

	if err = objectId.GetObjectId().SetWithSeq(seq); err != nil {
		if errors.Is(err, doddish.ErrUnsupportedSeq{}) {
			err = errors.BadRequest(err)
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	return err
}

// TODO support exact
func (objectId ObjectId) ContainsSku(
	objectGetter sku.TransactedGetter,
) (ok bool) {
	object := objectGetter.GetSku()

	metadata := object.GetMetadata()

	method := ids.Contains

	if objectId.Exact {
		method = ids.ContainsExactly
	}

	switch objectId.GetGenre() {

	case genres.Blob:
		purposeId := objectId.marklId.GetPurposeId()

		id := objects.GetMarklIdForPurpose(metadata, purposeId)

		return markl.Equals(objectId.marklId, id)

	case genres.Tag:
		if objectId.Exact {
			_, ok = metadata.GetIndex().GetTagPaths().All.ContainsObjectIdTagExact(
				objectId.GetObjectId(),
			)
		} else {
			_, ok = metadata.GetIndex().GetTagPaths().All.ContainsObjectIdTag(
				objectId.GetObjectId(),
			)
		}

		return ok

	case genres.Type:
		if method(metadata.GetType().ToType(), objectId.GetObjectId()) {
			return true
		}

		if transacted, isExternal := objectGetter.(*sku.Transacted); isExternal {
			if method(transacted.ExternalType, objectId.GetObjectId()) {
				return true
			}
		}
	}

	return method(&object.ObjectId, objectId.GetObjectId())
}

func (objectId ObjectId) String() string {
	var sb strings.Builder

	if objectId.Exact {
		sb.WriteRune('=')
	}

	if objectId.Virtual {
		sb.WriteRune('%')
	}

	sb.WriteString(ids.FormattedString(objectId.GetObjectId()))

	return sb.String()
}

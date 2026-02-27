package queries

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type exp struct {
	isObjectIds    bool
	expTagsOrTypes expTagsOrTypes
	expObjectIds   expObjectIds
}

func (e *exp) IsEmpty() bool {
	if len(e.expTagsOrTypes.Children) > 0 {
		return false
	}

	if !e.expObjectIds.IsEmpty() {
		return false
	}

	return true
}

func (e *exp) reduce(b *buildState) (err error) {
	if err = e.expTagsOrTypes.reduce(b); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

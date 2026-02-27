package sku

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func MakeErrMergeConflict(item *FSItem) (err *ErrMergeConflict) {
	err = &ErrMergeConflict{}

	if item != nil {
		err.ResetWith(item)
	}

	return err
}

type ErrMergeConflict struct {
	FSItem
}

func (err *ErrMergeConflict) Is(target error) bool {
	_, ok := target.(*ErrMergeConflict)
	return ok
}

func (err *ErrMergeConflict) Error() string {
	return fmt.Sprintf(
		"merge conflict for fds: Object: %q, Blob: %q",
		&err.Object,
		&err.Blob,
	)
}

func (err *ErrMergeConflict) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func IsErrMergeConflict(err error) bool {
	return errors.Is(err, &ErrMergeConflict{})
}

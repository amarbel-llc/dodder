package inventory_list_coders

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

type ErrAfterDecoding struct {
	err    error
	object *sku.Transacted // borrowed
}

func (err ErrAfterDecoding) Error() string {
	return fmt.Sprintf(
		"error after decoding: %s (object: %q)",
		err.err,
		sku.String(err.object),
	)
}

func (err ErrAfterDecoding) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrAfterDecoding) Is(target error) bool {
	if _, ok := target.(ErrAfterDecoding); ok {
		return true
	}

	return false
}

func (err ErrAfterDecoding) Unwrap() error {
	return err.err
}

package genres

import (
	"fmt"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/quiter_seq"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

func newPkgError(text string) pkgError {
	return errors.NewWithType[pkgErrDisamb](text)
}

func wrapAsPkgError(err error) pkgError {
	return errors.WrapWithType[pkgErrDisamb](err)
}

var ErrNoAbbreviation = newPkgError("no abbreviation")

func MakeErrUnsupportedGenre(g domain_interfaces.GenreGetter) error {
	return errors.WrapSkip(1, errUnsupportedGenre{Genre: g.GetGenre()})
}

func IsErrUnsupportedGenre(err error) bool {
	return errors.Is(err, errUnsupportedGenre{})
}

type errUnsupportedGenre struct {
	domain_interfaces.Genre
}

func (err errUnsupportedGenre) Is(target error) (ok bool) {
	_, ok = target.(errUnsupportedGenre)
	return ok
}

func (err errUnsupportedGenre) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err errUnsupportedGenre) Error() string {
	return fmt.Sprintf("unsupported genre: %q", err.Genre)
}

func MakeErrUnrecognizedGenre(v string) errUnrecognizedGenre {
	return errUnrecognizedGenre(v)
}

func IsErrUnrecognizedGenre(err error) bool {
	return errors.Is(err, errUnrecognizedGenre(""))
}

type errUnrecognizedGenre string

func (err errUnrecognizedGenre) Is(target error) (ok bool) {
	_, ok = target.(errUnrecognizedGenre)
	return ok
}

func (err errUnrecognizedGenre) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err errUnrecognizedGenre) Error() string {
	return fmt.Sprintf(
		"unknown genre: %q. Available genres: %q",
		string(err),
		slices.Collect(quiter_seq.Strings(slices.Values(All()))),
	)
}

type ErrWrongGenre struct {
	Expected, Actual Genre
}

func (err ErrWrongGenre) Is(target error) (ok bool) {
	_, ok = target.(ErrWrongGenre)
	return ok
}

func (err ErrWrongGenre) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrWrongGenre) Error() string {
	return fmt.Sprintf(
		"expected genre %q but got %q",
		err.Expected,
		err.Actual,
	)
}

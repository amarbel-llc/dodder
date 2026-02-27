package env_workspace

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/papa/store_workspace"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrUnsupportedType struct {
	Type ids.Type
}

func (err ErrUnsupportedType) Is(target error) bool {
	_, ok := target.(ErrUnsupportedType)
	return ok
}

func (err ErrUnsupportedType) Error() string {
	return fmt.Sprintf("unsupported type: %q", err.Type)
}

func (err ErrUnsupportedType) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func makeErrUnsupportedOperation(s *Store, op any) error {
	return ErrUnsupportedOperation{
		repoId:             s.RepoId,
		store:              s.StoreLike,
		operationInterface: op,
	}
}

type ErrUnsupportedOperation struct {
	repoId             ids.RepoId
	store              store_workspace.StoreLike
	operationInterface any
}

func (e ErrUnsupportedOperation) Error() string {
	return fmt.Sprintf(
		"store (%q:%T) does not support operation '%T'",
		e.repoId,
		e.store,
		e.operationInterface,
	)
}

func (e ErrUnsupportedOperation) Is(target error) bool {
	_, ok := target.(ErrUnsupportedOperation)
	return ok
}

func (e ErrUnsupportedOperation) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

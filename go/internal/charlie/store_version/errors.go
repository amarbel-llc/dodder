package store_version

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrFutureStoreVersion struct {
	domain_interfaces.StoreVersion
}

func (err ErrFutureStoreVersion) Error() string {
	return fmt.Sprintf(
		strings.Join(
			[]string{
				"store version is from the future: %q",
				"This means that this installation of dodder is likely out of date.",
			},
			". ",
		),
		err.StoreVersion,
	)
}

func (err ErrFutureStoreVersion) Is(target error) bool {
	_, ok := target.(ErrFutureStoreVersion)
	return ok
}

func (err ErrFutureStoreVersion) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

package markl_age_id

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type tipe struct{}

var _ domain_interfaces.MarklFormat = tipe{}

func (tipe tipe) GetMarklFormatId() string {
	return markl.FormatIdAgeX25519Sec
}

func (tipe tipe) GetSize() int {
	panic(errors.Err501NotImplemented)
}

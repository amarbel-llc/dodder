package markl_age_id

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type tipe struct{}

var _ domain_interfaces.MarklFormat = tipe{}

func (tipe tipe) GetMarklFormatId() string {
	return markl.FormatIdAgeX25519Sec
}

func (tipe tipe) GetSize() int {
	panic(errors.Err501NotImplemented)
}

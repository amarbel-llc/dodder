package markl_age_id

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type tipe struct{}

var _ mad_domain_interfaces.MarklFormat = tipe{}

func (tipe tipe) GetMarklFormatId() string {
	return markl.FormatIdAgeX25519Sec
}

func (tipe tipe) GetSize() int {
	panic(errors.Err501NotImplemented)
}

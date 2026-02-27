package objects

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type keyValues struct {
	SelfWithoutTai markl.Id // TODO move to a separate key-value store
}

func (index *index) GetSelfWithoutTai() domain_interfaces.MarklId {
	return &index.SelfWithoutTai
}

func (index *index) GetSelfWithoutTaiMutable() domain_interfaces.MarklIdMutable {
	return &index.SelfWithoutTai
}

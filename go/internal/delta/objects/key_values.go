package objects

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
)

type keyValues struct {
	SelfWithoutTai markl.Id // TODO move to a separate key-value store
}

func (index *index) GetSelfWithoutTai() mad_domain_interfaces.MarklId {
	return &index.SelfWithoutTai
}

func (index *index) GetSelfWithoutTaiMutable() mad_domain_interfaces.MarklIdMutable {
	return &index.SelfWithoutTai
}

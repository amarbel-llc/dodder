package sku

import (
	"encoding/gob"

	"code.linenisgreat.com/dodder/go/internal/_/external_state"
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

func init() {
	gob.Register(Transacted{})
}

type (
	Config interface {
		domain_interfaces.Config
		ids.InlineTypeChecker // TODO move out of konfig entirely
	}

	TransactedGetter interface {
		GetSku() *Transacted
	}

	ObjectWithList struct {
		Object, List *Transacted
	}

	ExternalLike interface {
		ids.ObjectIdGetter
		interfaces.Stringer
		TransactedGetter
		ExternalLikeGetter
		GetExternalState() external_state.State
		ExternalObjectIdGetter
		GetRepoId() ids.RepoId
	}

	ExternalLikeGetter interface {
		GetSkuExternal() *Transacted
	}
)

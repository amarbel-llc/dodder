package store_workspace

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

type (
	Store interface {
		GetObjectIdsForString(string) ([]domain_interfaces.ExternalObjectId, error)
	}

	StoreGetter interface {
		GetWorkspaceStoreForQuery(ids.RepoId) (Store, bool)
	}
)

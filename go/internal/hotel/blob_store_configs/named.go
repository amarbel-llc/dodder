package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/directory_layout"
)

type ConfigNamed struct {
	Path   directory_layout.BlobStorePath
	Config TypedConfig
}

func (configNamed ConfigNamed) GetId() blob_store_id.Id {
	return configNamed.Path.GetId()
}

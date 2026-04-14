package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
)

type ConfigNamed struct {
	Path   directory_layout.BlobStorePath
	Config TypedConfig
}

func (configNamed ConfigNamed) GetId() blob_store_id.Id {
	return configNamed.Path.GetId()
}

package genesis_configs

import (
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
)

type (
	TomlV2Common  = genesis_config_blobs.TomlV2Common
	TomlV2Private = genesis_config_blobs.TomlV2Private
	TomlV2Public  = genesis_config_blobs.TomlV2Public
)

var (
	_ ConfigPublic         = &TomlV2Public{}
	_ ConfigPrivate        = &TomlV2Private{}
	_ ConfigPrivateMutable = &TomlV2Private{}
)

package genesis_configs

import (
	genesis_config_blobs "code.linenisgreat.com/dodder/go/internal/bravo/genesis_config_blobs"
)

type (
	TomlV3Common  = genesis_config_blobs.TomlV3Common
	TomlV3Private = genesis_config_blobs.TomlV3Private
	TomlV3Public  = genesis_config_blobs.TomlV3Public
)

var (
	_ ConfigPublic         = &TomlV3Public{}
	_ ConfigPrivate        = &TomlV3Private{}
	_ ConfigPrivateMutable = &TomlV3Private{}
)

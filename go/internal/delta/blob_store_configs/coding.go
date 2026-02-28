package blob_store_configs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

// TODO transition to using this for all registrations instead of map literal
// below
func registerToml[CONFIG Config, CONFIG_PTR interface {
	ConfigMutable
	interfaces.Ptr[CONFIG]
}](
	typeMap triple_hyphen_io.CoderTypeMapWithoutType[Config],
	typeString string,
) struct{} {
	if existing, ok := typeMap[typeString]; ok {
		panic(
			fmt.Sprintf(
				"coder for type %q registered more than once! first registration: %#v",
				typeString,
				existing,
			),
		)
	}

	typeMap[typeString] = triple_hyphen_io.CoderToml[
		Config,
		*Config,
	]{
		Progenitor: func() Config {
			var config CONFIG
			return CONFIG_PTR(&config)
		},
	}

	return struct{}{}
}

var Coder = triple_hyphen_io.CoderToTypedBlob[Config]{
	Metadata: triple_hyphen_io.TypedMetadataCoder[Config]{},
	Blob: triple_hyphen_io.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlBlobStoreConfigV0: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV0{}
				},
			},
			ids.TypeTomlBlobStoreConfigV1: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV1{}
				},
			},
			ids.TypeTomlBlobStoreConfigV2: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV2{}
				},
			},
			ids.TypeTomlBlobStoreConfigSftpExplicitV0: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlSFTPV0{}
				},
			},
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0: triple_hyphen_io.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlSFTPViaSSHConfigV0{}
				},
			},
		},
	),
}

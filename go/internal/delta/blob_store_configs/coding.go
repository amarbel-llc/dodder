package blob_store_configs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

// TODO transition to using this for all registrations instead of map literal
// below
func registerToml[CONFIG Config, CONFIG_PTR interface {
	ConfigMutable
	interfaces.Ptr[CONFIG]
}](
	typeMap hyphence.CoderTypeMapWithoutType[Config],
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

	typeMap[typeString] = hyphence.CoderToml[
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

var Coder = hyphence.CoderToTypedBlob[Config]{
	Metadata: hyphence.TypedMetadataCoder[Config]{},
	Blob: hyphence.CoderTypeMapWithoutType[Config](
		map[string]interfaces.CoderBufferedReadWriter[*Config]{
			ids.TypeTomlBlobStoreConfigV0: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV0{}
				},
			},
			ids.TypeTomlBlobStoreConfigV1: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV1{}
				},
			},
			ids.TypeTomlBlobStoreConfigV2: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV2{}
				},
			},
			ids.TypeTomlBlobStoreConfigV3: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlV3{}
				},
			},
			ids.TypeTomlBlobStoreConfigSftpExplicitV0: hyphence.CoderToml[
				Config,
				*Config,
			]{
				Progenitor: func() Config {
					return &TomlSFTPV0{}
				},
			},
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0: hyphence.CoderToml[
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

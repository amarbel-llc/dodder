package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/0/options_tools"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	charlie_rc "code.linenisgreat.com/dodder/go/internal/charlie/repo_configs"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
)

type (
	Defaults            = charlie_rc.Defaults
	DefaultsGetter      = charlie_rc.DefaultsGetter
	DefaultsV0          = charlie_rc.DefaultsV0
	DefaultsV1          = charlie_rc.DefaultsV1
	DefaultsV1OmitEmpty = charlie_rc.DefaultsV1OmitEmpty
	V0                  = charlie_rc.V0
	V0Document          = charlie_rc.V0Document
	V1                  = charlie_rc.V1
	V1Document          = charlie_rc.V1Document
	V2                  = charlie_rc.V2
	V2Document          = charlie_rc.V2Document
)

var (
	DecodeV0                      = charlie_rc.DecodeV0
	DecodeV1                      = charlie_rc.DecodeV1
	DecodeV2                      = charlie_rc.DecodeV2
	DecodeDefaultsV1OmitEmptyInto = charlie_rc.DecodeDefaultsV1OmitEmptyInto
	EncodeDefaultsV1OmitEmptyFrom = charlie_rc.EncodeDefaultsV1OmitEmptyFrom
)

type (
	TypedBlob = hyphence.TypedBlob[ConfigOverlay]

	ConfigOverlay interface {
		DefaultsGetter
		file_extensions.OverlayGetter
		options_print.OverlayGetter
		GetToolOptions() options_tools.Options
	}

	ConfigOverlay2 interface {
		ConfigOverlay
		GetBlobStores() []blob_store_id.Id
		GetStreamIndexFixed() bool
	}
)

var (
	_ ConfigOverlay  = V0{}
	_ ConfigOverlay  = V1{}
	_ ConfigOverlay2 = V2{}
)

func Default(defaultType ids.Type) Config {
	return Config{
		DefaultType:    defaultType,
		DefaultTags:    ids.MakeTagSetFromSlice(),
		FileExtensions: file_extensions.Default(),
		PrintOptions:   options_print.DefaultOverlay().GetPrintOptionsOverlay(),
		ToolOptions: options_tools.Options{
			Merge: []string{
				"vimdiff",
			},
		},
	}
}

func DefaultOverlay(
	blobStores []blob_store_id.Id,
	defaultType ids.TypeStruct,
) TypedBlob {
	return TypedBlob{
		Type: ids.DefaultOrPanic(genres.Config).ToMadder(),
		Blob: V2{
			BlobStores: blobStores,
			Defaults: DefaultsV1{
				Type: defaultType,
				Tags: make([]ids.TagStruct, 0),
			},
			PrintOptions:   options_print.DefaultOverlay(),
			FileExtensions: file_extensions.DefaultOverlay(),
			Tools: options_tools.Options{
				Merge: []string{
					"vimdiff",
				},
			},
		},
	}
}

func GetBlobStores(
	config ConfigOverlay,
	otherwise []blob_store_id.Id,
) []blob_store_id.Id {
	if config, ok := config.(ConfigOverlay2); ok {
		return config.GetBlobStores()
	} else {
		return otherwise
	}
}

func GetStreamIndexFixed(
	config ConfigOverlay,
	otherwise bool,
) bool {
	if config, ok := config.(ConfigOverlay2); ok {
		return config.GetStreamIndexFixed()
	} else {
		return otherwise
	}
}

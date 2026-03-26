package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/options_print"
	"code.linenisgreat.com/dodder/go/internal/_/options_tools"
	"code.linenisgreat.com/dodder/go/internal/alfa/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

type (
	TypedBlob = hyphence.TypedBlob[ConfigOverlay]

	DefaultsGetter interface {
		GetDefaults() Defaults
	}

	ConfigOverlay interface {
		DefaultsGetter
		file_extensions.OverlayGetter
		options_print.OverlayGetter
		GetToolOptions() options_tools.Options
	}

	ConfigOverlay2 interface {
		ConfigOverlay
		GetBlobStores() []blob_store_id.Id
	}

	Defaults interface {
		GetDefaultType() ids.TypeStruct
		GetDefaultTags() collections_slice.Slice[ids.TagStruct]
	}
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
		Type: ids.DefaultOrPanic(genres.Config),
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

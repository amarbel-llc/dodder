package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/_/options_print"
	"code.linenisgreat.com/dodder/go/internal/_/options_tools"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
)

type V2 struct {
	DefaultBlobStoreId blob_store_id.Id       `toml:"default-blob_store"`
	Defaults           DefaultsV1             `toml:"defaults"`
	FileExtensions     file_extensions.TOMLV1 `toml:"file-extensions"`
	PrintOptions       options_print.V2       `toml:"cli-output"`
	Tools              options_tools.Options  `toml:"tools"`
}

var _ ConfigOverlay2 = V2{}

func (config *V2) Reset() {
	config.FileExtensions.Reset()
	config.Defaults.Type = ids.TypeStruct{}
	config.Defaults.Tags = make([]ids.TagStruct, 0)
	config.PrintOptions = options_print.V2{}
}

func (config *V2) ResetWith(b *V2) {
	config.FileExtensions.Reset()

	config.Defaults.Type = b.Defaults.Type

	config.Defaults.Tags = make([]ids.TagStruct, len(b.Defaults.Tags))
	copy(config.Defaults.Tags, b.Defaults.Tags)

	config.PrintOptions = b.PrintOptions
}

func (config V2) GetDefaults() Defaults {
	return config.Defaults
}

func (config V2) GetFileExtensionsOverlay() file_extensions.Overlay {
	return config.FileExtensions.GetFileExtensionsOverlay()
}

func (config V2) GetPrintOptionsOverlay() options_print.Overlay {
	return config.PrintOptions.GetPrintOptionsOverlay()
}

func (config V2) GetToolOptions() options_tools.Options {
	return config.Tools
}

func (config V2) GetDefaultBlobStoreId() blob_store_id.Id {
	return config.DefaultBlobStoreId
}

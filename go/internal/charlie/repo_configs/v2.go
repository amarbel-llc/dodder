package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/0/options_tools"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_id"
)

//go:generate tommy generate
type V2 struct {
	BlobStores       []blob_store_id.Id     `toml:"blob-stores"`
	Defaults         DefaultsV1             `toml:"defaults"`
	FileExtensions   file_extensions.TOMLV1 `toml:"file-extensions"`
	PrintOptions     options_print.V2       `toml:"cli-output"`
	Tools            options_tools.Options  `toml:"tools"`
	StreamIndexFixed bool                   `toml:"stream-index-fixed,omitempty"`
}

func (config *V2) Reset() {
	config.BlobStores = make([]blob_store_id.Id, 0)
	config.FileExtensions.Reset()
	config.Defaults.Type = ids.TypeStruct{}
	config.Defaults.Tags = make([]ids.TagStruct, 0)
	config.PrintOptions = options_print.V2{}
	config.StreamIndexFixed = false
}

func (config *V2) ResetWith(b *V2) {
	config.BlobStores = make([]blob_store_id.Id, len(b.BlobStores))
	copy(config.BlobStores, b.BlobStores)

	config.FileExtensions.Reset()

	config.Defaults.Type = b.Defaults.Type

	config.Defaults.Tags = make([]ids.TagStruct, len(b.Defaults.Tags))
	copy(config.Defaults.Tags, b.Defaults.Tags)

	config.PrintOptions = b.PrintOptions

	config.StreamIndexFixed = b.StreamIndexFixed
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

func (config V2) GetBlobStores() []blob_store_id.Id {
	return config.BlobStores
}

func (config V2) GetStreamIndexFixed() bool {
	return config.StreamIndexFixed
}

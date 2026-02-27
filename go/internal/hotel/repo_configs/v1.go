package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/options_tools"
	"code.linenisgreat.com/dodder/go/internal/delta/options_print"
	"code.linenisgreat.com/dodder/go/internal/echo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

type DefaultsV1 struct {
	Type ids.TypeStruct  `toml:"type,omitempty"`
	Tags []ids.TagStruct `toml:"tags"`
}

var _ ConfigOverlay = V1{}

func (defaults DefaultsV1) GetDefaultType() ids.TypeStruct {
	return defaults.Type
}

func (defaults DefaultsV1) GetDefaultTags() collections_slice.Slice[ids.TagStruct] {
	return collections_slice.Slice[ids.TagStruct](defaults.Tags)
}

type DefaultsV1OmitEmpty struct {
	Type ids.TypeStruct  `toml:"type,omitempty"`
	Tags []ids.TagStruct `toml:"tags,omitempty"`
}

func (defaults DefaultsV1OmitEmpty) GetDefaultType() ids.TypeStruct {
	return defaults.Type
}

func (defaults DefaultsV1OmitEmpty) GetDefaultTags() collections_slice.Slice[ids.TagStruct] {
	return collections_slice.Slice[ids.TagStruct](defaults.Tags)
}

type V1 struct {
	Defaults       DefaultsV1             `toml:"defaults"`
	FileExtensions file_extensions.TOMLV1 `toml:"file-extensions"`
	PrintOptions   options_print.V1       `toml:"cli-output"`
	Tools          options_tools.Options  `toml:"tools"`
}

var _ ConfigOverlay = V1{}

func (blob *V1) Reset() {
	blob.FileExtensions.Reset()
	blob.Defaults.Type = ids.TypeStruct{}
	blob.Defaults.Tags = make([]ids.TagStruct, 0)
	blob.PrintOptions = options_print.V1{}
}

func (blob *V1) ResetWith(b *V1) {
	blob.FileExtensions.Reset()

	blob.Defaults.Type = b.Defaults.Type

	blob.Defaults.Tags = make([]ids.TagStruct, len(b.Defaults.Tags))
	copy(blob.Defaults.Tags, b.Defaults.Tags)

	blob.PrintOptions = b.PrintOptions
}

func (blob V1) GetDefaults() Defaults {
	return blob.Defaults
}

func (blob V1) GetFileExtensionsOverlay() file_extensions.Overlay {
	return blob.FileExtensions.GetFileExtensionsOverlay()
}

func (blob V1) GetPrintOptionsOverlay() options_print.Overlay {
	return blob.PrintOptions.GetPrintOptionsOverlay()
}

func (blob V1) GetToolOptions() options_tools.Options {
	return blob.Tools
}

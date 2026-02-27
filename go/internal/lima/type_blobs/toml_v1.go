package type_blobs

import (
	"code.linenisgreat.com/dodder/go/internal/_/reset"
	"code.linenisgreat.com/dodder/go/internal/charlie/script_config"
)

type TomlV1 struct {
	Binary        bool                                      `toml:"binary,omitempty"`
	FileExtension string                                    `toml:"file-extension,omitempty"`
	MimeType      string                                    `toml:"mime-type,omitempty"`
	ExecCommand   *script_config.ScriptConfig               `toml:"exec-command,omitempty"`
	VimSyntaxType string                                    `toml:"vim-syntax-type"`
	UTIGroups     map[string]UTIGroup                       `toml:"uti-groups"`
	Formatters    map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`

	// TODO migrate to properly-typed hooks
	Hooks any `toml:"hooks"`
}

func (blob *TomlV1) Reset() {
	blob.Binary = false
	blob.FileExtension = ""
	blob.MimeType = ""
	blob.ExecCommand = nil
	blob.VimSyntaxType = ""

	blob.UTIGroups = reset.Map(blob.UTIGroups)
	blob.Formatters = reset.Map(blob.Formatters)
	blob.Hooks = nil
}

func (blob *TomlV1) GetBinary() bool {
	return blob.Binary
}

func (blob *TomlV1) GetFileExtension() string {
	return blob.FileExtension
}

func (blob *TomlV1) GetMimeType() string {
	return blob.MimeType
}

func (blob *TomlV1) GetVimSyntaxType() string {
	return blob.VimSyntaxType
}

func (blob *TomlV1) GetFormatters() map[string]script_config.WithOutputFormat {
	return blob.Formatters
}

func (blob *TomlV1) GetFormatterUTIGroups() map[string]UTIGroup {
	return blob.UTIGroups
}

func (blob *TomlV1) GetStringLuaHooks() string {
	hooks, _ := blob.Hooks.(string)
	return hooks
}

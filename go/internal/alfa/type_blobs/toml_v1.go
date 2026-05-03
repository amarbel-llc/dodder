package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/0/reset"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
)

//go:generate tommy generate
type TomlV1 struct {
	Binary        bool                                      `toml:"binary,omitempty"`
	FileExtension string                                    `toml:"file-extension,omitempty"`
	MimeType      string                                    `toml:"mime-type,omitempty"`
	ExecCommand   *script_config.ScriptConfig               `toml:"exec-command,omitempty"`
	VimSyntaxType string                                    `toml:"vim-syntax-type"`
	UTIGroups     map[string]UTIGroup                       `toml:"uti-groups"`
	Formatters    map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`

	Hooks      string            `toml:"hooks"`
	References *ReferencesConfig `toml:"references,omitempty"`
}

func (blob *TomlV1) Reset() {
	blob.Binary = false
	blob.FileExtension = ""
	blob.MimeType = ""
	blob.ExecCommand = nil
	blob.VimSyntaxType = ""

	blob.UTIGroups = reset.Map(blob.UTIGroups)
	blob.Formatters = reset.Map(blob.Formatters)
	blob.Hooks = ""
	blob.References = nil
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
	return blob.Hooks
}

func (blob *TomlV1) GetReferences() *ReferencesConfig {
	return blob.References
}

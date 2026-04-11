package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/0/reset"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

//go:generate tommy generate
type TomlV2 struct {
	Binary        bool                                      `toml:"binary,omitempty"`
	FileExtension string                                    `toml:"file-extension,omitempty"`
	MimeType      string                                    `toml:"mime-type,omitempty"`
	ExecCommand   *script_config.ScriptConfig               `toml:"exec-command,omitempty"`
	VimSyntaxType string                                    `toml:"vim-syntax-type"`
	UTIGroups     map[string]UTIGroup                       `toml:"uti-groups"`
	Formatters    map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`

	Hooks      string            `toml:"hooks"`
	References *ReferencesConfig `toml:"references,omitempty"`

	Fields       []FieldDefinition           `toml:"fields,omitempty"`
	FieldsReader *script_config.ScriptConfig `toml:"fields-reader,omitempty"`
	FieldsWriter *script_config.ScriptConfig `toml:"fields-writer,omitempty"`
}

func (blob *TomlV2) Reset() {
	blob.Binary = false
	blob.FileExtension = ""
	blob.MimeType = ""
	blob.ExecCommand = nil
	blob.VimSyntaxType = ""

	blob.UTIGroups = reset.Map(blob.UTIGroups)
	blob.Formatters = reset.Map(blob.Formatters)
	blob.Hooks = ""
	blob.References = nil

	blob.Fields = nil
	blob.FieldsReader = nil
	blob.FieldsWriter = nil
}

func (blob *TomlV2) GetBinary() bool {
	return blob.Binary
}

func (blob *TomlV2) GetFileExtension() string {
	return blob.FileExtension
}

func (blob *TomlV2) GetMimeType() string {
	return blob.MimeType
}

func (blob *TomlV2) GetVimSyntaxType() string {
	return blob.VimSyntaxType
}

func (blob *TomlV2) GetFormatters() map[string]script_config.WithOutputFormat {
	return blob.Formatters
}

func (blob *TomlV2) GetFormatterUTIGroups() map[string]UTIGroup {
	return blob.UTIGroups
}

func (blob *TomlV2) GetStringLuaHooks() string {
	return blob.Hooks
}

func (blob *TomlV2) GetReferences() *ReferencesConfig {
	return blob.References
}

func (blob *TomlV2) GetFieldDefinitions() []FieldDefinition {
	return blob.Fields
}

func (blob *TomlV2) GetFieldsReader() *script_config.ScriptConfig {
	return blob.FieldsReader
}

func (blob *TomlV2) GetFieldsWriter() *script_config.ScriptConfig {
	return blob.FieldsWriter
}

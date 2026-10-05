package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/0/reset"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
)

// TomlV3 carries the same fields as TomlV2. What changes is the meaning of
// Hooks: a v3 type's lua hook script receives the English-keyed object tables
// (Genre/ObjectId/Type/Tags, sku_lua.LuaTableV2) where v0-v2 scripts receive
// the German-keyed ones (Gattung/Kennung/Typ/Etiketten). See
// GetLuaHookTableVersion and dodder#407.
//
//go:generate tommy generate
type TomlV3 struct {
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

func (blob *TomlV3) Reset() {
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

func (blob *TomlV3) GetBinary() bool {
	return blob.Binary
}

func (blob *TomlV3) GetFileExtension() string {
	return blob.FileExtension
}

func (blob *TomlV3) GetMimeType() string {
	return blob.MimeType
}

func (blob *TomlV3) GetVimSyntaxType() string {
	return blob.VimSyntaxType
}

func (blob *TomlV3) GetFormatters() map[string]script_config.WithOutputFormat {
	return blob.Formatters
}

func (blob *TomlV3) GetFormatterUTIGroups() map[string]UTIGroup {
	return blob.UTIGroups
}

func (blob *TomlV3) GetStringLuaHooks() string {
	return blob.Hooks
}

func (blob *TomlV3) GetLuaHookTableVersion() LuaHookTableVersion {
	return LuaHookTableV2
}

func (blob *TomlV3) GetReferences() *ReferencesConfig {
	return blob.References
}

func (blob *TomlV3) GetFieldDefinitions() []FieldDefinition {
	return blob.Fields
}

func (blob *TomlV3) GetFieldsReader() *script_config.ScriptConfig {
	return blob.FieldsReader
}

func (blob *TomlV3) GetFieldsWriter() *script_config.ScriptConfig {
	return blob.FieldsWriter
}

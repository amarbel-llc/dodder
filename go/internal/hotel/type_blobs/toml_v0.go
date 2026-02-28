package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/_/reset"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

type TomlV0 struct {
	InlineBlob    bool                        `toml:"inline-akte,omitempty"`
	Archived      bool                        `toml:"archived,omitempty"`
	FileExtension string                      `toml:"file-extension,omitempty"`
	ExecCommand   *script_config.ScriptConfig `toml:"exec-command,omitempty"`
	VimSyntaxType string                      `toml:"vim-syntax-type"`
	// TODO-P4 rename to uti-groups
	FormatterUTIGroups map[string]UTIGroup                       `toml:"formatter-uti-groups"`
	Formatters         map[string]script_config.WithOutputFormat `toml:"formatters,omitempty"`
	Actions            map[string]script_config.ScriptConfig     `toml:"actions,omitempty"`
	Hooks              interface{}                               `toml:"hooks"`
}

func (blob *TomlV0) Reset() {
	blob.Archived = false
	blob.InlineBlob = false
	blob.FileExtension = ""
	blob.ExecCommand = nil
	blob.VimSyntaxType = ""

	blob.FormatterUTIGroups = reset.Map(blob.FormatterUTIGroups)
	blob.Formatters = reset.Map(blob.Formatters)
	blob.Actions = reset.Map(blob.Actions)
	blob.Hooks = nil
}

func (blob *TomlV0) GetBinary() bool {
	return !blob.InlineBlob
}

func (blob *TomlV0) GetFileExtension() string {
	return blob.FileExtension
}

func (blob *TomlV0) GetMimeType() string {
	return ""
}

func (blob *TomlV0) GetVimSyntaxType() string {
	return blob.VimSyntaxType
}

func (blob *TomlV0) GetFormatters() map[string]script_config.WithOutputFormat {
	return blob.Formatters
}

func (blob *TomlV0) GetFormatterUTIGroups() map[string]UTIGroup {
	return blob.FormatterUTIGroups
}

func (blob *TomlV0) GetStringLuaHooks() string {
	hooks, _ := blob.Hooks.(string)
	return hooks
}

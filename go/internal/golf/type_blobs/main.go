package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

func Default() TomlV2 {
	return TomlV2{
		FileExtension: "md",
		VimSyntaxType: "markdown",
	}
}

func DefaultWithPandocFormatter() TomlV2 {
	return TomlV2{
		FileExtension: "md",
		Formatters: map[string]script_config.WithOutputFormat{
			"text": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Normalize markdown with pandoc",
					Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit`,
				},
				FileExtension: "md",
			},
		},
		VimSyntaxType: "markdown",
	}
}

func DefaultPandocDefaults() TomlV2 {
	return TomlV2{
		FileExtension: "yaml",
		Formatters:    make(map[string]script_config.WithOutputFormat),
	}
}

func DefaultPandocLuaFilter() TomlV2 {
	return TomlV2{
		FileExtension: "lua",
		Formatters:    make(map[string]script_config.WithOutputFormat),
	}
}

type Blob interface {
	GetFileExtension() string
	GetBinary() bool
	GetMimeType() string
	GetVimSyntaxType() string

	WithFormatters
	WithFormatterUTIGroups
	WithStringLuaHooks
	WithReferences
}

type WithFormatters interface {
	GetFormatters() map[string]script_config.WithOutputFormat
}

type WithFormatterUTIGroups interface {
	GetFormatterUTIGroups() map[string]UTIGroup
}

// TODO make typed hooks
type WithStringLuaHooks interface {
	GetStringLuaHooks() string
}

type WithReferences interface {
	GetReferences() *ReferencesConfig
}

type WithFields interface {
	GetFieldDefinitions() []FieldDefinition
	GetFieldsReader() *script_config.ScriptConfig
	GetFieldsWriter() *script_config.ScriptConfig
}

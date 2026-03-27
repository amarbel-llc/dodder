package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

func Default() TomlV1 {
	return TomlV1{
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

func DefaultPandocDefaults() TomlV1 {
	return TomlV1{
		FileExtension: "yaml",
		Formatters:    make(map[string]script_config.WithOutputFormat),
	}
}

func DefaultPandocLuaFilter() TomlV1 {
	return TomlV1{
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

var (
	_ Blob = &TomlV0{}
	_ Blob = &TomlV1{}
)

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

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

// actionableFields is the field set shared by !task and !chore. Both built-in
// types use the same status/priority/due triple. Future work
// (see docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md §1a)
// is to extract this into an !actionable abstract type that both compose
// against.
func actionableFields() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:    "status",
			Kind:    "enum",
			Values:  []string{"todo", "in_progress", "done", "cancelled"},
			Default: "todo",
		},
		{
			Name:    "priority",
			Kind:    "enum",
			Values:  []string{"p0", "p1", "p2", "p3"},
			Default: "p3",
		},
		{
			Name: "due",
			Kind: "string",
		},
	}
}

// actionableFieldsReader returns the yq script that projects fields from a
// TOML blob into Metadata.Index.Fields during commit. The output JSON keys
// must match the field names declared by actionableFields.
func actionableFieldsReader() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o json '{"status": .status, "priority": .priority, "due": .due}'`,
	}
}

// actionableFieldsWriter returns the yq script that projects field edits back
// into the TOML blob during organize mutations. Reads DODDER_FIELD_status,
// DODDER_FIELD_priority, DODDER_FIELD_due env vars and writes them into the
// blob at DODDER_BLOB_PATH.
func actionableFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\"" "$DODDER_BLOB_PATH"`,
	}
}

// DefaultTaskType returns the built-in !task type blob, used by genesis when
// BigBang.IncludeBuiltinActionableTypes is set. !task instances are stored as
// TOML blobs mirroring the field values; the reader script projects them into
// the index on commit, the writer script projects user edits back into the
// blob during organize mutations. The CalDAV haustoria emits these blobs
// directly during compile.
func DefaultTaskType() TomlV2 {
	return TomlV2{
		FileExtension: "toml",
		VimSyntaxType: "toml",
		Fields:        actionableFields(),
		FieldsReader:  actionableFieldsReader(),
		FieldsWriter:  actionableFieldsWriter(),
	}
}

// DefaultChoreType returns the built-in !chore type blob. Same field set as
// !task; calendar-to-type binding stays a workspace config concern (the
// CalDAV haustoria's tasks calendar binds to !task and chores binds to
// !chore). Future !actionable abstract type will replace the duplication.
func DefaultChoreType() TomlV2 {
	return TomlV2{
		FileExtension: "toml",
		VimSyntaxType: "toml",
		Fields:        actionableFields(),
		FieldsReader:  actionableFieldsReader(),
		FieldsWriter:  actionableFieldsWriter(),
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

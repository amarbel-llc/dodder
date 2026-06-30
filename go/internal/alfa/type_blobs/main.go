package type_blobs

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
)

// ArchiveTag is the tag the built-in actionable hooks add to archive an object
// on a terminal status. Genesis seeds it into the dormant index (when the
// built-in actionable types are included) so a carrying object becomes
// dormant. The "zz-" prefix follows the repo's archive-tag convention and
// keeps it sorted last.
const ArchiveTag = "zz-archive"

// actionableArchiveHook is the shared on_commit_fields lua hook for the
// built-in actionable types. Branching on the projected status field
// (kinder.Fields.status), populated by the on_commit_fields commit stage
// before invocation:
//
//   - status == "cancelled": archive (add ArchiveTag) for every actionable
//     type. Tag-only, no field mutation, so no write-back fires.
//   - status == "done" on a !task: one-shot completion -> archive. Tag-only.
//   - status == "done" on a recurring type (!chore / !habit) carrying a
//     non-empty recurrence: roll the object forward instead of archiving --
//     advance kinder.Fields.due by the recurrence duration (via the host
//     dodder_advance_date helper) and reset status to "todo". Mutating the
//     `due` / `status` fields triggers the RFC 0006 Phase 1 commit-time field
//     write-back (a single bounded, hook-free tryWriteFields + tryReadFields
//     pass), persisting the recurred values to the blob.
//
// An empty `due` is guarded: a recurring task with no date only resets status,
// leaving due empty (nothing to advance). A recurring "done" with no
// recurrence value falls through unchanged.
func actionableArchiveHook() string {
	return fmt.Sprintf(`return {
  on_commit_fields = function(kinder, mutter)
    local f = kinder.Fields
    if not f then return end
    local status = f.status
    if status == "cancelled" then
      kinder.Etiketten[%[1]q] = true
    elseif status == "done" then
      if kinder.Typ == "!task" then
        kinder.Etiketten[%[1]q] = true
      elseif f.recurrence ~= nil and f.recurrence ~= "" then
        if f.due ~= nil and f.due ~= "" then
          f.due = dodder_advance_date(f.due, f.recurrence)
        end
        f.status = "todo"
      end
    end
  end,
}
`, ArchiveTag)
}

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

// actionableFields is the one-shot field set used by !task. Recurring
// actionable types (!chore, !habit) extend it with a recurrence field via
// recurringFields. Future work
// (see docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md §1a)
// is to extract this into an !actionable abstract type that both compose
// against. The urgency field is left without a default so an untriaged
// instance reads as urgency-unset rather than silently defaulting.
func actionableFields() []FieldDefinition {
	return []FieldDefinition{
		{
			Name:    "status",
			Kind:    "enum",
			Values:  []string{"todo", "in_progress", "done", "cancelled"},
			Default: "todo",
		},
		{
			Name:   "urgency",
			Kind:   "enum",
			Values: []string{"0_hour", "1_day", "2_week", "3_month", "4_quarter", "5_episode", "6_year"},
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

// recurringFields is the field set used by recurring actionable types (!chore,
// !habit): the actionable triple plus a recurrence cadence. recurrence is an
// ISO-8601 duration string (e.g. "P1W" for weekly); kept as a plain string
// rather than a dedicated date kind since there is no date FieldDefinition
// kind.
func recurringFields() []FieldDefinition {
	return append(actionableFields(), FieldDefinition{
		Name: "recurrence",
		Kind: "string",
	})
}

// actionableFieldsReader returns the yq script that projects fields from a
// TOML blob into Metadata.Index.Fields during commit. The output JSON keys
// must match the field names declared by actionableFields. Null-valued keys
// (absent in the blob) are dropped via with_entries(select(.value != null))
// so an unset optional field (e.g. urgency) reads as field-unset rather than
// being projected as an explicit null that commit would reject; this also
// restores the field-level defaults for status/priority when absent.
func actionableFieldsReader() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due} | with_entries(select(.value != null))'`,
	}
}

// actionableFieldsWriter returns the yq script that projects field edits back
// into the TOML blob during organize mutations. Reads DODDER_FIELD_status,
// DODDER_FIELD_urgency, DODDER_FIELD_priority, DODDER_FIELD_due env vars and
// writes them into the blob at DODDER_BLOB_PATH.
func actionableFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .urgency = \"$DODDER_FIELD_urgency\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\"" "$DODDER_BLOB_PATH"`,
	}
}

// recurringFieldsReader mirrors actionableFieldsReader but also projects the
// recurrence field declared by recurringFields. It drops null-valued keys for
// the same reason: an unset optional field reads as field-unset rather than a
// commit-rejected explicit null.
func recurringFieldsReader() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o json '{"status": .status, "urgency": .urgency, "priority": .priority, "due": .due, "recurrence": .recurrence} | with_entries(select(.value != null))'`,
	}
}

// recurringFieldsWriter mirrors actionableFieldsWriter but also writes the
// recurrence field from DODDER_FIELD_recurrence into the blob.
func recurringFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i ".status = \"$DODDER_FIELD_status\" | .urgency = \"$DODDER_FIELD_urgency\" | .priority = \"$DODDER_FIELD_priority\" | .due = \"$DODDER_FIELD_due\" | .recurrence = \"$DODDER_FIELD_recurrence\"" "$DODDER_BLOB_PATH"`,
	}
}

// actionableFormatters renders the dang-typed `body` field via the
// blob-backed pandoc tooling (materialized to $DODDER_BLOB_TREE). The body is
// extracted from the TOML blob, the leading `#!dang ...` convention line is
// stripped (Phase-1 stand-in until the dang mechanism lands, see issue #296),
// then normalized with the shared dodder-edit defaults.
func actionableFormatters() map[string]script_config.WithOutputFormat {
	return map[string]script_config.WithOutputFormat{
		"text": {
			ScriptConfig: script_config.ScriptConfig{
				Description: "Render the dang-typed body with pandoc",
				Script:      `yq -p toml -r '.body' | sed '1{/^#!dang/d}' | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit`,
			},
			FileExtension: "md",
		},
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
		Formatters:    actionableFormatters(),
		Hooks:         actionableArchiveHook(),
		Fields:        actionableFields(),
		FieldsReader:  actionableFieldsReader(),
		FieldsWriter:  actionableFieldsWriter(),
	}
}

// DefaultChoreType returns the built-in !chore type blob. !chore is a
// recurring actionable type, so it carries the recurrence field on top of the
// actionable triple; calendar-to-type binding stays a workspace config concern
// (the CalDAV haustoria's tasks calendar binds to !task and chores binds to
// !chore). Future !actionable abstract type will replace the duplication.
func DefaultChoreType() TomlV2 {
	return TomlV2{
		FileExtension: "toml",
		VimSyntaxType: "toml",
		Formatters:    actionableFormatters(),
		Hooks:         actionableArchiveHook(),
		Fields:        recurringFields(),
		FieldsReader:  recurringFieldsReader(),
		FieldsWriter:  recurringFieldsWriter(),
	}
}

// DefaultHabitType returns the built-in !habit type blob. It is structurally
// identical to !chore (the recurring field set + recurring reader/writer) and
// shares the actionable recurrence hook; the semantic distinction (a
// consistency practice vs a periodic obligation) surfaces in the per-instance
// recurrence cadence and the type description, not in the field schema.
func DefaultHabitType() TomlV2 {
	return TomlV2{
		FileExtension: "toml",
		VimSyntaxType: "toml",
		Formatters:    actionableFormatters(),
		Hooks:         actionableArchiveHook(),
		Fields:        recurringFields(),
		FieldsReader:  recurringFieldsReader(),
		FieldsWriter:  recurringFieldsWriter(),
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

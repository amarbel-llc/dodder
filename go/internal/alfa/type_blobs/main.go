package type_blobs

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
)

// ArchiveTag is the tag the built-in actionable hooks add to archive an object
// on a terminal status. Genesis seeds it into the dormant index (when the
// built-in actionable types are included) so a carrying object becomes
// dormant. The "zz-" prefix follows the repo's archive-tag convention and
// keeps it sorted last.
//
// COUPLING: the "zz-archive" string literal in the blob-backed
// actionable-common.lua module
// (romeo/local_working_copy/embedded/actionable/actionable-common.lua) MUST
// match this const. The archive logic lives in that lua module now (delivered
// as a blob reference on the actionable type objects); this const is still
// used by genesis to seed the dormant index.
const ArchiveTag = "zz-archive"

// actionableCommonHook is the thin type-blob hook script for the built-in
// actionable types: it require()s the blob-backed actionable-common module
// (delivered as a blob reference on the type object, preloaded into the hook
// VM by oscar/store) and returns its hooks table. The archive/recurrence/
// completed-date logic lives in embedded/actionable/actionable-common.lua.
func actionableCommonHook() string {
	return `local common = require("actionable-common")
return common.hooks
`
}

func Default() TomlV2 {
	return TomlV2{
		FileExtension: "md",
		VimSyntaxType: "markdown",
	}
}

// pandocBeamerScript wraps pandoc's beamer PDF writer in a fifo so the binary
// PDF can still stream to the formatter's stdout: pandoc refuses to write PDF
// output to stdout, so the script mints a fifo, drains it to stdout with a
// background cat, and points pandoc's --output at the fifo. Pandoc only opens
// the output file after the LaTeX engine succeeds, so on failure the fifo is
// opened once for write to unblock the drain before exiting nonzero.
func pandocBeamerScript() string {
	return `tmp="$(mktemp -d)" || exit 1
trap 'rm -rf "$tmp"' EXIT
mkfifo "$tmp/out.pdf" || exit 1
cat "$tmp/out.pdf" &
if ! pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-beamer --output="$tmp/out.pdf"; then
  : >"$tmp/out.pdf"
  wait
  exit 1
fi
wait`
}

// DefaultWithPandocFormatter is the builtin pandoc-backed !md type blob. Its
// formatters split into two pipelines: the EDIT pipeline (dodder-edit.lua:
// typed code blocks are normalized inline as text — safe for checkout/editing
// and for any type, used by text/html/html-gdoc/pdf-beamer) and the RENDER
// pipeline (dodder-render.lua: typed code blocks are replaced with rendered
// images via `dodder format-object -stdin png <type>`, so the referenced type
// must ship a png formatter — used by text-render/html-partial). The
// output-flavored html/html-gdoc formatters deliberately stay on the edit
// filter: builtin types ship no png formatter, so the render filter would
// hard-fail any document embedding a builtin-typed code block, and its
// image-file side effects don't fit stdout-pipe formatters.
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
			"text-render": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Render markdown for output with pandoc (typed code blocks become images)",
					Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-render`,
				},
				FileExtension: "md",
			},
			"html": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Render markdown to an HTML fragment with pandoc",
					Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-html`,
				},
				FileExtension: "html",
			},
			"html-partial": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Render markdown to an HTML fragment via the render pipeline (typed code blocks become images)",
					Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-html-partial`,
				},
				FileExtension: "html",
			},
			"html-gdoc": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Render markdown to standalone HTML for pasting into Google Docs",
					Script:      `pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-gdoc`,
				},
				FileExtension: "html",
			},
			"pdf-beamer": {
				ScriptConfig: script_config.ScriptConfig{
					Description: "Render markdown to a beamer slide PDF (requires a host LaTeX engine)",
					Script:      pandocBeamerScript(),
				},
				FileExtension: "pdf",
			},
		},
		// UTI groups bundle formatters by output medium so `format-object
		// -uti-group <name>` (and UTI-aware consumers) can pick the right
		// formatter for a requested UTI. Every value must name a formatter in
		// Formatters above.
		UTIGroups: map[string]UTIGroup{
			"default": {
				"public.utf8-plain-text": "text",
				"public.html":            "html",
			},
			"text-render": {
				"public.utf8-plain-text": "text-render",
				"public.html":            "html",
			},
			"gdoc": {
				"public.utf8-plain-text": "text",
				"public.html":            "html-gdoc",
			},
			"pdf": {
				"public.utf8-plain-text": "text",
				"com.adobe.pdf":          "pdf-beamer",
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
//
// Values are read via yq's strenv() env accessor rather than shell-interpolated
// into the expression, so a field value containing a double-quote (or yq
// expression syntax) is treated as string data, not expression text. The
// free-form `due`/`recurrence` fields make this a live injection surface; the
// enum-constrained fields are safe by construction but use strenv() uniformly.
// strenv (not env) forces the value to a string, matching the enum/string field
// kinds. The expression is single-quoted so the shell performs no substitution
// (see #297).
func actionableFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i '.status = strenv(DODDER_FIELD_status) | .urgency = strenv(DODDER_FIELD_urgency) | .priority = strenv(DODDER_FIELD_priority) | .due = strenv(DODDER_FIELD_due)' "$DODDER_BLOB_PATH"`,
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
// recurrence field from DODDER_FIELD_recurrence into the blob. Like
// actionableFieldsWriter, values are read via yq's strenv() accessor so a
// value containing a double-quote is treated as string data, not expression
// text; recurrence is a free-form string, so it is a live injection surface
// (see #297).
func recurringFieldsWriter() *script_config.ScriptConfig {
	return &script_config.ScriptConfig{
		Script: `yq -p toml -o toml -i '.status = strenv(DODDER_FIELD_status) | .urgency = strenv(DODDER_FIELD_urgency) | .priority = strenv(DODDER_FIELD_priority) | .due = strenv(DODDER_FIELD_due) | .recurrence = strenv(DODDER_FIELD_recurrence)' "$DODDER_BLOB_PATH"`,
	}
}

// actionableBodyExtract is the shared pipeline prefix for the actionable body
// formatters: it extracts the dang-typed `body` field from the TOML blob and
// strips the leading `#!dang ...` convention line (Phase-1 stand-in until the
// dang mechanism lands, see issue #296).
const actionableBodyExtract = `yq -p toml -r '.body' | sed '1{/^#!dang/d}'`

// actionableFormatters renders the dang-typed `body` field via the
// blob-backed pandoc tooling (materialized to $DODDER_BLOB_TREE): text
// normalizes with the shared dodder-edit defaults, html/html-gdoc mirror the
// !md formatters of the same name. No pdf-beamer here: slides don't fit task
// prose.
func actionableFormatters() map[string]script_config.WithOutputFormat {
	return map[string]script_config.WithOutputFormat{
		"text": {
			ScriptConfig: script_config.ScriptConfig{
				Description: "Render the dang-typed body with pandoc",
				Script:      actionableBodyExtract + ` | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-edit`,
			},
			FileExtension: "md",
		},
		"html": {
			ScriptConfig: script_config.ScriptConfig{
				Description: "Render the dang-typed body to an HTML fragment with pandoc",
				Script:      actionableBodyExtract + ` | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-html`,
			},
			FileExtension: "html",
		},
		"html-gdoc": {
			ScriptConfig: script_config.ScriptConfig{
				Description: "Render the dang-typed body to standalone HTML for pasting into Google Docs",
				Script:      actionableBodyExtract + ` | pandoc --data-dir="$DODDER_BLOB_TREE" --defaults=dodder-gdoc`,
			},
			FileExtension: "html",
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
		Hooks:         actionableCommonHook(),
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
		Hooks:         actionableCommonHook(),
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
		Hooks:         actionableCommonHook(),
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

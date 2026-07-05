//go:build test

package type_blobs

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestDefaultWithPandocFormatter(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultWithPandocFormatter()

	t.AssertEqualStrings("md", blob.FileExtension)
	t.AssertEqualStrings("markdown", blob.VimSyntaxType)

	assertFormatterSet(&t, blob.Formatters, map[string]string{
		"text":       "md",
		"html":       "html",
		"html-gdoc":  "html",
		"pdf-beamer": "pdf",
	})

	// Every formatter must resolve pandoc tooling from the materialized blob
	// tree, never from host pandoc data dirs (portability contract).
	for name, formatter := range blob.Formatters {
		if !strings.Contains(formatter.Script, `--data-dir="$DODDER_BLOB_TREE"`) {
			t.Errorf("formatter %q script missing blob-tree data-dir: %q", name, formatter.Script)
		}
	}

	// pandoc refuses to write PDF to stdout, so pdf-beamer must route the
	// output through a fifo drained to stdout.
	if !strings.Contains(blob.Formatters["pdf-beamer"].Script, "mkfifo") {
		t.Errorf("pdf-beamer script missing fifo wrapper: %q", blob.Formatters["pdf-beamer"].Script)
	}
}

func TestDefaultTaskType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultTaskType()

	t.AssertEqualStrings("toml", blob.FileExtension)
	t.AssertEqualStrings("toml", blob.VimSyntaxType)

	assertActionableFormatters(&t, blob.Formatters)

	assertActionableFields(&t, blob.Fields)

	// !task is the one-shot actionable type: it must NOT carry recurrence.
	for _, f := range blob.Fields {
		if f.Name == "recurrence" {
			t.Errorf("DefaultTaskType must not have a recurrence field, got %v", blob.Fields)
		}
	}

	t.AssertNotNil(blob.FieldsReader, "FieldsReader")

	if !strings.Contains(blob.FieldsReader.Script, "yq -p toml -o json") {
		t.Errorf("FieldsReader script missing yq invocation: %q", blob.FieldsReader.Script)
	}

	if !strings.Contains(blob.FieldsReader.Script, `"urgency": .urgency`) {
		t.Errorf("FieldsReader script missing urgency projection: %q", blob.FieldsReader.Script)
	}

	// The reader must drop null-valued keys so an unset optional field (e.g.
	// urgency) reads as field-unset rather than a commit-rejected explicit null.
	if !strings.Contains(blob.FieldsReader.Script, "with_entries(select(.value != null))") {
		t.Errorf("FieldsReader script missing null-key drop: %q", blob.FieldsReader.Script)
	}

	t.AssertNotNil(blob.FieldsWriter, "FieldsWriter")

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_FIELD_status") {
		t.Errorf("FieldsWriter script missing DODDER_FIELD_status: %q", blob.FieldsWriter.Script)
	}

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_FIELD_urgency") {
		t.Errorf("FieldsWriter script missing DODDER_FIELD_urgency: %q", blob.FieldsWriter.Script)
	}

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_BLOB_PATH") {
		t.Errorf("FieldsWriter script missing DODDER_BLOB_PATH: %q", blob.FieldsWriter.Script)
	}

	// The archive/recurrence/completed-date logic now lives in the blob-backed
	// actionable-common.lua module; the type Hooks string is a thin loader that
	// require()s it (delivered as a blob reference on the type object,
	// preloaded into the hook VM by oscar/store).
	if !strings.Contains(blob.Hooks, `require("actionable-common")`) {
		t.Errorf("Hooks missing actionable-common require: %q", blob.Hooks)
	}
}

func TestDefaultChoreType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultChoreType()

	t.AssertEqualStrings("toml", blob.FileExtension)

	assertActionableFormatters(&t, blob.Formatters)

	assertRecurringFields(&t, blob.Fields)

	t.AssertNotNil(blob.FieldsReader, "FieldsReader")

	if !strings.Contains(blob.FieldsReader.Script, `"recurrence": .recurrence`) {
		t.Errorf("FieldsReader script missing recurrence projection: %q", blob.FieldsReader.Script)
	}

	if !strings.Contains(blob.FieldsReader.Script, "with_entries(select(.value != null))") {
		t.Errorf("FieldsReader script missing null-key drop: %q", blob.FieldsReader.Script)
	}

	t.AssertNotNil(blob.FieldsWriter, "FieldsWriter")

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_FIELD_recurrence") {
		t.Errorf("FieldsWriter script missing DODDER_FIELD_recurrence: %q", blob.FieldsWriter.Script)
	}

	// !chore shares the thin actionable-common loader with !task; the
	// recurrence/archive branching lives in the blob-backed
	// actionable-common.lua module, not the inline Hooks string.
	if !strings.Contains(blob.Hooks, `require("actionable-common")`) {
		t.Errorf("Hooks missing actionable-common require: %q", blob.Hooks)
	}
}

func TestDefaultHabitType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultHabitType()

	t.AssertEqualStrings("toml", blob.FileExtension)
	t.AssertEqualStrings("toml", blob.VimSyntaxType)

	assertActionableFormatters(&t, blob.Formatters)

	assertRecurringFields(&t, blob.Fields)

	// !habit is structurally identical to !chore (recurring field set +
	// recurring reader/writer); the distinction lives in the hooks, not the
	// schema.
	choreBlob := DefaultChoreType()

	t.AssertNotNil(blob.FieldsReader, "FieldsReader")
	t.AssertNotNil(blob.FieldsWriter, "FieldsWriter")

	t.AssertEqualStrings(choreBlob.FieldsReader.Script, blob.FieldsReader.Script)
	t.AssertEqualStrings(choreBlob.FieldsWriter.Script, blob.FieldsWriter.Script)
}

// assertFormatterSet asserts formatters contains exactly the names in
// expected, each with the expected output file extension.
func assertFormatterSet(
	t *ui.T,
	formatters map[string]script_config.WithOutputFormat,
	expected map[string]string,
) {
	if len(formatters) != len(expected) {
		t.Errorf(
			"expected %d formatters (%v), got %d (%v)",
			len(expected), expected, len(formatters), formatters,
		)
	}

	for name, fileExtension := range expected {
		formatter, ok := formatters[name]

		if !ok {
			t.Errorf("missing formatter %q", name)
			continue
		}

		t.AssertEqualStrings(fileExtension, formatter.FileExtension)

		if formatter.Script == "" {
			t.Errorf("formatter %q has an empty script", name)
		}
	}
}

// assertActionableFormatters asserts the shared actionable formatter set
// (!task/!chore/!habit): text/html/html-gdoc render the dang-typed body via
// blob-backed pandoc; pdf-beamer is deliberately absent (slides don't fit
// task prose).
func assertActionableFormatters(
	t *ui.T,
	formatters map[string]script_config.WithOutputFormat,
) {
	assertFormatterSet(t, formatters, map[string]string{
		"text":      "md",
		"html":      "html",
		"html-gdoc": "html",
	})

	for name, formatter := range formatters {
		if !strings.HasPrefix(formatter.Script, `yq -p toml -r '.body'`) {
			t.Errorf("formatter %q script missing body extraction prefix: %q", name, formatter.Script)
		}

		if !strings.Contains(formatter.Script, `--data-dir="$DODDER_BLOB_TREE"`) {
			t.Errorf("formatter %q script missing blob-tree data-dir: %q", name, formatter.Script)
		}
	}
}

type expectedField struct {
	name      string
	kind      string
	values    []string
	dflt      string
	hasValues bool
}

func actionableExpectedFields() []expectedField {
	return []expectedField{
		{
			name:      "status",
			kind:      "enum",
			values:    []string{"todo", "in_progress", "done", "cancelled"},
			dflt:      "todo",
			hasValues: true,
		},
		{
			name:      "urgency",
			kind:      "enum",
			values:    []string{"0_hour", "1_day", "2_week", "3_month", "4_quarter", "5_episode", "6_year"},
			dflt:      "",
			hasValues: true,
		},
		{
			name:      "priority",
			kind:      "enum",
			values:    []string{"p0", "p1", "p2", "p3"},
			dflt:      "p3",
			hasValues: true,
		},
		{
			name:      "due",
			kind:      "string",
			hasValues: false,
		},
	}
}

func assertActionableFields(t *ui.T, fields []FieldDefinition) {
	expected := actionableExpectedFields()

	t.AssertLen(len(expected), fields, "fields")

	assertFields(t, expected, fields)
}

func assertRecurringFields(t *ui.T, fields []FieldDefinition) {
	expected := append(actionableExpectedFields(), expectedField{
		name:      "recurrence",
		kind:      "string",
		hasValues: false,
	})

	t.AssertLen(len(expected), fields, "fields")

	assertFields(t, expected, fields)
}

func assertFields(t *ui.T, expected []expectedField, fields []FieldDefinition) {
	for i, want := range expected {
		got := fields[i]

		t.AssertEqualStrings(want.name, got.Name)
		t.AssertEqualStrings(want.kind, got.Kind)
		t.AssertEqualStrings(want.dflt, got.Default)

		if want.hasValues {
			if len(got.Values) != len(want.values) {
				t.Fatalf("field %d (%s): expected %d values, got %d", i, want.name, len(want.values), len(got.Values))
			}

			for j := range want.values {
				t.AssertEqualStrings(want.values[j], got.Values[j])
			}
		} else {
			if len(got.Values) != 0 {
				t.Errorf("field %d (%s): expected no values, got %v", i, want.name, got.Values)
			}
		}
	}
}

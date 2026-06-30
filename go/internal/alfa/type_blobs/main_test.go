//go:build test

package type_blobs

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestDefaultTaskType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultTaskType()

	t.AssertEqualStrings("toml", blob.FileExtension)
	t.AssertEqualStrings("toml", blob.VimSyntaxType)

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

	// !task archives on terminal status via the on_commit_fields hook: both
	// "cancelled" and "done" add the archive tag (the done branch is gated on
	// !task).
	if !strings.Contains(blob.Hooks, "on_commit_fields") {
		t.Errorf("Hooks missing on_commit_fields: %q", blob.Hooks)
	}

	if !strings.Contains(blob.Hooks, ArchiveTag) {
		t.Errorf("Hooks missing archive tag %q: %q", ArchiveTag, blob.Hooks)
	}

	if !strings.Contains(blob.Hooks, `if kinder.Typ == "!task" then`) {
		t.Errorf("Hooks missing !task done branch: %q", blob.Hooks)
	}
}

func TestDefaultChoreType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultChoreType()

	t.AssertEqualStrings("toml", blob.FileExtension)

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

	// !chore archives on "cancelled" but on "done" recurs instead of archiving:
	// the shared hook gates the archive on kinder.Typ == "!task" and, for a
	// recurring type carrying a recurrence, advances due and resets status to
	// "todo" via the dodder_advance_date host helper.
	if !strings.Contains(blob.Hooks, "on_commit_fields") {
		t.Errorf("Hooks missing on_commit_fields: %q", blob.Hooks)
	}

	if !strings.Contains(blob.Hooks, `if kinder.Typ == "!task" then`) {
		t.Errorf("Hooks missing !task-gated archive branch: %q", blob.Hooks)
	}

	if !strings.Contains(blob.Hooks, "dodder_advance_date(f.due, f.recurrence)") {
		t.Errorf("Hooks missing recurrence advance branch: %q", blob.Hooks)
	}
}

func TestDefaultHabitType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultHabitType()

	t.AssertEqualStrings("toml", blob.FileExtension)
	t.AssertEqualStrings("toml", blob.VimSyntaxType)

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

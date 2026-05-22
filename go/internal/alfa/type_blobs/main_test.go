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

	t.AssertNotNil(blob.FieldsReader, "FieldsReader")

	if !strings.Contains(blob.FieldsReader.Script, "yq -p toml -o json") {
		t.Errorf("FieldsReader script missing yq invocation: %q", blob.FieldsReader.Script)
	}

	t.AssertNotNil(blob.FieldsWriter, "FieldsWriter")

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_FIELD_status") {
		t.Errorf("FieldsWriter script missing DODDER_FIELD_status: %q", blob.FieldsWriter.Script)
	}

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_BLOB_PATH") {
		t.Errorf("FieldsWriter script missing DODDER_BLOB_PATH: %q", blob.FieldsWriter.Script)
	}
}

func TestDefaultChoreType(t1 *testing.T) {
	t := ui.MakeT(t1)

	blob := DefaultChoreType()

	t.AssertEqualStrings("toml", blob.FileExtension)

	assertActionableFields(&t, blob.Fields)

	// chore and task currently share the exact same field set + scripts;
	// this is enforced by both calling actionableFields/Reader/Writer.
	taskBlob := DefaultTaskType()

	t.AssertEqualStrings(taskBlob.FieldsReader.Script, blob.FieldsReader.Script)
	t.AssertEqualStrings(taskBlob.FieldsWriter.Script, blob.FieldsWriter.Script)
}

func assertActionableFields(t *ui.T, fields []FieldDefinition) {
	t.AssertLen(3, fields, "fields")

	expected := []struct {
		name      string
		kind      string
		values    []string
		dflt      string
		hasValues bool
	}{
		{
			name:      "status",
			kind:      "enum",
			values:    []string{"todo", "in_progress", "done", "cancelled"},
			dflt:      "todo",
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

//go:build test

package type_blobs

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

func TestDefaultTaskType(t1 *testing.T) {
	t := ui.T{T: t1}

	blob := DefaultTaskType()

	if blob.FileExtension != "toml" {
		t.Errorf("expected file-extension %q, got %q", "toml", blob.FileExtension)
	}

	if blob.VimSyntaxType != "toml" {
		t.Errorf("expected vim-syntax-type %q, got %q", "toml", blob.VimSyntaxType)
	}

	assertActionableFields(t, blob.Fields)

	if blob.FieldsReader == nil {
		t.Fatalf("expected non-nil FieldsReader")
	}

	if !strings.Contains(blob.FieldsReader.Script, "yq -p toml -o json") {
		t.Errorf("FieldsReader script missing yq invocation: %q", blob.FieldsReader.Script)
	}

	if blob.FieldsWriter == nil {
		t.Fatalf("expected non-nil FieldsWriter")
	}

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_FIELD_status") {
		t.Errorf("FieldsWriter script missing DODDER_FIELD_status: %q", blob.FieldsWriter.Script)
	}

	if !strings.Contains(blob.FieldsWriter.Script, "DODDER_BLOB_PATH") {
		t.Errorf("FieldsWriter script missing DODDER_BLOB_PATH: %q", blob.FieldsWriter.Script)
	}
}

func TestDefaultChoreType(t1 *testing.T) {
	t := ui.T{T: t1}

	blob := DefaultChoreType()

	if blob.FileExtension != "toml" {
		t.Errorf("expected file-extension %q, got %q", "toml", blob.FileExtension)
	}

	assertActionableFields(t, blob.Fields)

	// chore and task currently share the exact same field set + scripts;
	// this is enforced by both calling actionableFields/Reader/Writer.
	taskBlob := DefaultTaskType()

	if blob.FieldsReader.Script != taskBlob.FieldsReader.Script {
		t.Errorf("chore and task FieldsReader scripts diverged")
	}

	if blob.FieldsWriter.Script != taskBlob.FieldsWriter.Script {
		t.Errorf("chore and task FieldsWriter scripts diverged")
	}
}

func assertActionableFields(t ui.T, fields []FieldDefinition) {
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

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

		if got.Name != want.name {
			t.Errorf("field %d: expected name %q, got %q", i, want.name, got.Name)
		}

		if got.Kind != want.kind {
			t.Errorf("field %d: expected kind %q, got %q", i, want.kind, got.Kind)
		}

		if got.Default != want.dflt {
			t.Errorf("field %d: expected default %q, got %q", i, want.dflt, got.Default)
		}

		if want.hasValues {
			if len(got.Values) != len(want.values) {
				t.Fatalf("field %d (%s): expected %d values, got %d", i, want.name, len(want.values), len(got.Values))
			}

			for j := range want.values {
				if got.Values[j] != want.values[j] {
					t.Errorf("field %d (%s) value %d: expected %q, got %q", i, want.name, j, want.values[j], got.Values[j])
				}
			}
		} else {
			if len(got.Values) != 0 {
				t.Errorf("field %d (%s): expected no values, got %v", i, want.name, got.Values)
			}
		}
	}
}

//go:build test

package type_blobs

import (
	"strings"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestValidateTerminalAcceptsSubsetOfValues(t1 *testing.T) {
	t := ui.MakeT(t1)

	fd := FieldDefinition{
		Name:     "status",
		Kind:     "enum",
		Values:   []string{"todo", "done", "cancelled"},
		Terminal: []string{"done", "cancelled"},
	}

	if err := fd.ValidateTerminal(); err != nil {
		t.Fatalf("expected no error, got %s", err)
	}
}

func TestValidateTerminalRejectsValueOutsideValues(t1 *testing.T) {
	t := ui.MakeT(t1)

	fd := FieldDefinition{
		Name:     "status",
		Kind:     "enum",
		Values:   []string{"todo", "done"},
		Terminal: []string{"archived"},
	}

	err := fd.ValidateTerminal()

	if err == nil || !strings.Contains(err.Error(), `terminal value "archived"`) {
		t.Errorf("expected terminal-value error, got %v", err)
	}
}

func TestValidateTerminalRejectsNonEnumField(t1 *testing.T) {
	t := ui.MakeT(t1)

	fd := FieldDefinition{
		Name:     "due",
		Kind:     "string",
		Terminal: []string{"2026-01-01"},
	}

	err := fd.ValidateTerminal()

	if err == nil || !strings.Contains(err.Error(), "only valid on enum fields") {
		t.Errorf("expected enum-only error, got %v", err)
	}
}

func TestBuiltinActionableTerminalStatuses(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, testCase := range []struct {
		name     string
		blob     TomlV3
		terminal []string
	}{
		{"task", DefaultTaskType(), []string{"done", "cancelled"}},
		{"chore", DefaultChoreType(), []string{"cancelled"}},
		{"habit", DefaultHabitType(), []string{"cancelled"}},
	} {
		status := testCase.blob.GetFieldDefinitions()[0]

		t.AssertEqualStrings("status", status.Name)
		t.AssertEqualStrings(
			strings.Join(testCase.terminal, ","),
			strings.Join(status.Terminal, ","),
		)

		if err := status.ValidateTerminal(); err != nil {
			t.Errorf("%s: %s", testCase.name, err)
		}
	}
}

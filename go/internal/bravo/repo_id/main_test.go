package repo_id

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestParse(t1 *testing.T) {
	t := ui.MakeT(t1)

	type expectation struct {
		input     string
		empty     bool
		cwd       bool
		system    bool
		name      string
		cwdDepth  uint
		canonical string
	}

	expectations := []expectation{
		{input: "", empty: true, canonical: ""},
		{input: ".", cwd: true, name: "", canonical: "."},
		{input: "/", system: true, name: "", canonical: "/"},
		{input: "work", name: "work", canonical: "work"},
		{input: "~work", name: "work", canonical: "work"},
		{input: ".notes", cwd: true, name: "notes", canonical: ".notes"},
		{input: "..notes", cwd: true, name: "notes", cwdDepth: 1, canonical: ".notes"},
		{input: "//backup", system: true, name: "backup", canonical: "//backup"},
		{input: "/backup", system: true, name: "backup", canonical: "//backup"},
	}

	for _, e := range expectations {
		var id Id

		if err := id.Set(e.input); err != nil {
			t.Errorf("Set(%q) returned error: %s", e.input, err)
			continue
		}

		if id.IsEmpty() != e.empty {
			t.Errorf("Set(%q): IsEmpty = %v, want %v", e.input, id.IsEmpty(), e.empty)
		}

		if id.IsCwd() != e.cwd {
			t.Errorf("Set(%q): IsCwd = %v, want %v", e.input, id.IsCwd(), e.cwd)
		}

		if id.IsSystem() != e.system {
			t.Errorf("Set(%q): IsSystem = %v, want %v", e.input, id.IsSystem(), e.system)
		}

		if id.GetName() != e.name {
			t.Errorf("Set(%q): GetName = %q, want %q", e.input, id.GetName(), e.name)
		}

		if id.GetCwdDepth() != e.cwdDepth {
			t.Errorf("Set(%q): GetCwdDepth = %d, want %d", e.input, id.GetCwdDepth(), e.cwdDepth)
		}

		if id.String() != e.canonical {
			t.Errorf("Set(%q): String = %q, want %q", e.input, id.String(), e.canonical)
		}
	}
}

func TestParseErrors(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, input := range []string{"..", "work/sub", "wo rk", "wo@rk"} {
		var id Id

		if err := id.Set(input); err == nil {
			t.Errorf("Set(%q): expected error, got none", input)
		}
	}
}

func TestCheckPrototypeSupported(t1 *testing.T) {
	t := ui.MakeT(t1)

	var nearest Id
	if err := nearest.Set(".notes"); err != nil {
		t.Fatalf("Set(.notes): %s", err)
	}
	if err := nearest.CheckPrototypeSupported(); err != nil {
		t.Errorf(".notes should be supported, got: %s", err)
	}

	var deeper Id
	if err := deeper.Set("..notes"); err != nil {
		t.Fatalf("Set(..notes): %s", err)
	}
	if err := deeper.CheckPrototypeSupported(); err == nil {
		t.Errorf("..notes should be rejected by the prototype")
	}
}

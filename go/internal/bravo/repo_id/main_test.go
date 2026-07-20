package repo_id

import (
	"testing"

	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestIsAuto(t1 *testing.T) {
	t := ui.MakeT(t1)

	// The zero value (neither -repo_id nor DODDER_REPO_ID given) is auto.
	var zero scoped_id.Id
	if !IsAuto(zero) {
		t.Errorf("IsAuto(zero) = false, want true")
	}

	// The cwd default and every explicit selection are NOT auto.
	if IsAuto(CwdDefault()) {
		t.Errorf("IsAuto(CwdDefault()) = true, want false")
	}

	for _, value := range []string{"work", ".notes", "//backup"} {
		var id scoped_id.Id

		if err := id.Set(value); err != nil {
			t.Fatalf("Set(%q): %s", value, err)
		}

		if IsAuto(id) {
			t.Errorf("IsAuto(Set(%q)) = true, want false", value)
		}
	}
}

func TestEffectiveName(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Auto and the cwd default both resolve to the fixed default name.
	var zero scoped_id.Id
	if got := EffectiveName(zero); got != DefaultName {
		t.Errorf("EffectiveName(zero) = %q, want %q", got, DefaultName)
	}

	if got := EffectiveName(CwdDefault()); got != DefaultName {
		t.Errorf("EffectiveName(CwdDefault()) = %q, want %q", got, DefaultName)
	}

	for value, want := range map[string]string{
		"work":     "work",
		".notes":   "notes",
		"//backup": "backup",
	} {
		var id scoped_id.Id

		if err := id.Set(value); err != nil {
			t.Fatalf("Set(%q): %s", value, err)
		}

		if got := EffectiveName(id); got != want {
			t.Errorf("EffectiveName(Set(%q)) = %q, want %q", value, got, want)
		}
	}
}

func TestCheckSupported(t1 *testing.T) {
	t := ui.MakeT(t1)

	// The cwd default, single-dot / user names, multi-dot cwd depth (#281),
	// and the forced-system `//name` spelling all resolve. Bare `/` is the
	// nameless forced-system selector (not remote-first), so it resolves too.
	if err := CheckSupported(CwdDefault()); err != nil {
		t.Errorf("CheckSupported(CwdDefault()) = %s, want nil", err)
	}

	for _, value := range []string{"work", ".notes", "..notes", "...notes", "//backup", "/"} {
		var id scoped_id.Id

		if err := id.Set(value); err != nil {
			t.Fatalf("Set(%q): %s", value, err)
		}

		if err := CheckSupported(id); err != nil {
			t.Errorf("CheckSupported(Set(%q)) = %s, want nil", value, err)
		}
	}

	// Still gated: only the remote-first `/name` spelling (no remote transport).
	for _, value := range []string{"/backup"} {
		var id scoped_id.Id

		if err := id.Set(value); err != nil {
			t.Fatalf("Set(%q): %s", value, err)
		}

		if err := CheckSupported(id); err == nil {
			t.Errorf("CheckSupported(Set(%q)) = nil, want error", value)
		}
	}
}

package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupIndex redirects XDG_STATE_HOME at a fresh temp dir so each test gets an
// isolated index — the same sandbox-by-XDG-redirection the bats lanes rely on.
func setupIndex(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

// makeRepo creates a repo dir with a config-seed file in it and returns
// (base, configPath).
func makeRepo(t *testing.T) (base, config string) {
	t.Helper()
	base = t.TempDir()
	config = filepath.Join(base, "config-seed")
	if err := os.WriteFile(config, []byte("x"), 0o444); err != nil {
		t.Fatal(err)
	}
	return base, config
}

func TestKeyIsCleanedDeterministicAndDistinct(t *testing.T) {
	a := Key("/tmp/repo")
	if Key("/tmp/repo/") != a || Key("/tmp/./repo") != a {
		t.Fatal("Key is not path-cleaned before hashing")
	}
	if len(a) != 16 {
		t.Fatalf("Key length = %d, want 16 hex chars", len(a))
	}
	if Key("/tmp/other") == a {
		t.Fatal("distinct paths produced the same key")
	}
}

func TestIndexDirUnderXDGState(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	want := filepath.Join(state, "dodder", "index")
	if got := IndexDir(); got != want {
		t.Fatalf("IndexDir() = %q, want %q", got, want)
	}
}

func TestRegisterWritesSymlinkAtKeyPointingAtConfig(t *testing.T) {
	setupIndex(t)
	base, config := makeRepo(t)

	if err := Register(base, config); err != nil {
		t.Fatalf("Register: %v", err)
	}

	link := filepath.Join(IndexDir(), Key(base))
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("expected a symlink at the key: %v", err)
	}
	if target != config {
		t.Fatalf("symlink target = %q, want %q", target, config)
	}
}

func TestRegisterIsIdempotentReplace(t *testing.T) {
	setupIndex(t)
	base, config := makeRepo(t)

	if err := Register(base, config); err != nil {
		t.Fatal(err)
	}
	if err := Register(base, config); err != nil {
		t.Fatalf("re-register: %v", err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("re-registering one repo produced %d entries, want 1", len(entries))
	}
}

func TestEntriesClassifiesLiveAndDangling(t *testing.T) {
	setupIndex(t)

	liveBase, liveConfig := makeRepo(t)
	if err := Register(liveBase, liveConfig); err != nil {
		t.Fatal(err)
	}

	deadBase, deadConfig := makeRepo(t)
	if err := Register(deadBase, deadConfig); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(deadBase); err != nil {
		t.Fatal(err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	byKey := map[string]Entry{}
	for _, e := range entries {
		byKey[e.Key] = e
	}
	if e := byKey[Key(liveBase)]; e.Dangling {
		t.Error("live repo classified as dangling")
	}
	if e := byKey[Key(deadBase)]; !e.Dangling {
		t.Error("deleted repo not classified as dangling")
	}
	// RepoDir recovers the base even for the dangling entry.
	if got := byKey[Key(deadBase)].RepoDir(); got != deadBase {
		t.Errorf("RepoDir() = %q, want %q", got, deadBase)
	}
}

func TestEntriesSkipsTempAndNonSymlinks(t *testing.T) {
	setupIndex(t)
	base, config := makeRepo(t)
	if err := Register(base, config); err != nil {
		t.Fatal(err)
	}

	dir := IndexDir()
	// A leftover staging file and a stray regular file must both be ignored.
	if err := os.WriteFile(filepath.Join(dir, ".tmp-99-99"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "not-a-symlink"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (temp + regular file must be skipped)", len(entries))
	}
}

func TestShouldPrune(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	old := now.Add(-48 * time.Hour)
	fresh := now.Add(-1 * time.Minute)
	retention := 24 * time.Hour

	cases := []struct {
		name      string
		dangling  bool
		mtime     time.Time
		retention time.Duration
		want      bool
	}{
		{"aged dangling past retention", true, old, retention, true},
		{"fresh dangling within grace", true, fresh, retention, false},
		{"live is never pruned", false, old, retention, false},
		{"zero retention is a no-op", true, old, 0, false},
		{"negative retention is a no-op", true, old, -time.Hour, false},
	}
	for _, c := range cases {
		if got := shouldPrune(c.dangling, c.mtime, now, c.retention); got != c.want {
			t.Errorf("%s: shouldPrune = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestGCZeroRetentionIsNoOp(t *testing.T) {
	setupIndex(t)
	base, config := makeRepo(t)
	if err := Register(base, config); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(base); err != nil {
		t.Fatal(err)
	}

	removed, err := GC(0)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("GC(0) removed %d, want 0 (no-op)", removed)
	}
	entries, _ := Entries()
	if len(entries) != 1 {
		t.Fatalf("GC(0) mutated the index: %d entries left, want 1", len(entries))
	}
}

func TestGCKeepsLiveAndFreshDangling(t *testing.T) {
	setupIndex(t)

	liveBase, liveConfig := makeRepo(t)
	if err := Register(liveBase, liveConfig); err != nil {
		t.Fatal(err)
	}

	freshDeadBase, freshDeadConfig := makeRepo(t)
	if err := Register(freshDeadBase, freshDeadConfig); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(freshDeadBase); err != nil {
		t.Fatal(err)
	}

	// A generous retention: the just-registered dangling entry is within the
	// grace window and the live one is never a candidate, so nothing is pruned.
	removed, err := GC(DefaultRetention())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("GC pruned %d fresh/live entries, want 0", removed)
	}
	entries, _ := Entries()
	if len(entries) != 2 {
		t.Fatalf("GC removed live/fresh entries: %d left, want 2", len(entries))
	}
}

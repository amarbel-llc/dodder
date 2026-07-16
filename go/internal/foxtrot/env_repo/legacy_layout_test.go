//go:build test

package env_repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Pins the #363 distinct-error-state behavior: a scope root with a legacy
// (pre-FDR-0019) flat-layout config-seed is detected and reported
// differently from a scope root with no config-seed at all.
func TestDetectLegacyRepoLayout_FindsLegacyTree(t1 *testing.T) {
	t := ui.MakeT(t1)

	scopeRoot := t1.TempDir()

	if err := os.WriteFile(filepath.Join(scopeRoot, "config-seed"), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("failed to write fixture config-seed: %v", err)
	}

	nestedDataDir := filepath.Join(scopeRoot, "repos", "default")

	legacyErr, ok := detectLegacyRepoLayout(nestedDataDir)
	if !ok {
		t.Fatalf("expected a legacy layout to be detected at %q", scopeRoot)
	}

	if legacyErr.ScopeRoot != scopeRoot {
		t.Fatalf("expected ScopeRoot %q, got %q", scopeRoot, legacyErr.ScopeRoot)
	}

	if legacyErr.Name != "default" {
		t.Fatalf("expected Name %q, got %q", "default", legacyErr.Name)
	}
}

func TestDetectLegacyRepoLayout_NoLegacyTree(t1 *testing.T) {
	t := ui.MakeT(t1)

	scopeRoot := t1.TempDir()
	nestedDataDir := filepath.Join(scopeRoot, "repos", "default")

	if _, ok := detectLegacyRepoLayout(nestedDataDir); ok {
		t.Fatalf("expected no legacy layout to be detected at %q", scopeRoot)
	}
}

// A non-nested path (no "repos" parent component) should never be reported
// as a legacy tree -- the shape check itself is the safety gate before any
// stat happens.
func TestDetectLegacyRepoLayout_NonNestedPathNeverMatches(t1 *testing.T) {
	t := ui.MakeT(t1)

	scopeRoot := t1.TempDir()

	if err := os.WriteFile(filepath.Join(scopeRoot, "config-seed"), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("failed to write fixture config-seed: %v", err)
	}

	if _, ok := detectLegacyRepoLayout(scopeRoot); ok {
		t.Fatalf("expected no legacy layout to be detected for a non-nested path %q", scopeRoot)
	}
}

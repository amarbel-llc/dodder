package mcp_dodder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func renderSystemPromptAppendBody(t *ui.T, data systemPromptAppendData) string {
	var b strings.Builder
	t.AssertNoError(promptsTmpl.ExecuteTemplate(&b, "system-prompt-append", data))
	return b.String()
}

func TestSystemPromptAppendUserScopeWithCountsAndRepos(t1 *testing.T) {
	t := ui.MakeT(t1)

	got := renderSystemPromptAppendBody(&t, systemPromptAppendData{
		Version:         "0.2.7+abc1234",
		BoundRepo:       "default",
		Scope:           "XDG-user scope (spelled name)",
		HasWorkspace:    false,
		CountsAvailable: true,
		TypeCount:       5,
		TagCount:        12,
		RepoCount:       2,
		Repos:           []string{"default", "work"},
	})

	expected := `# dodder repository (bound to this MCP server)

Server: dodder 0.2.7+abc1234 (this is the binary serving this MCP — match it against a bug report's version before assuming a bug is unfixed).
Bound repo: default — XDG-user scope (spelled name).
No workspace here — store operations only.
Indexed: 5 type(s), 12 tag(s).

2 repo(s) addressable from here (each usable as a -repo_id):
  - default
  - work

Next: use the discover prompt to map the type/tag vocabulary, or read dodder:///repos to switch repos.
`

	if got != expected {
		t.Fatalf("system-prompt-append body mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, expected)
	}
}

func TestSystemPromptAppendCwdScopeWorkspaceNoCountsNoRepos(t1 *testing.T) {
	t := ui.MakeT(t1)

	got := renderSystemPromptAppendBody(&t, systemPromptAppendData{
		Version:         "dev+unknown",
		BoundRepo:       ".notes",
		Scope:           "cwd-ancestor .dodder scope (spelled .name)",
		HasWorkspace:    true,
		CountsAvailable: false,
		RepoCount:       0,
		Repos:           nil,
	})

	expected := `# dodder repository (bound to this MCP server)

Server: dodder dev+unknown (this is the binary serving this MCP — match it against a bug report's version before assuming a bug is unfixed).
Bound repo: .notes — cwd-ancestor .dodder scope (spelled .name).
A workspace is active (a checked-out working copy is present).

0 repo(s) addressable from here.

Next: use the discover prompt to map the type/tag vocabulary, or read dodder:///repos to switch repos.
`

	if got != expected {
		t.Fatalf("system-prompt-append body mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, expected)
	}
}

// TestScopedReposBothScopes verifies the enumeration shared by the
// dodder:///repos listing and the system-prompt-append fragment: the active
// cwd scope is spelled `.name`, the XDG-user scope `name`, non-directories
// are ignored, and the result is sorted by repo_id.
func TestScopedReposBothScopes(t1 *testing.T) {
	t := ui.MakeT(t1)

	cwdRepos := t1.TempDir()
	userRepos := t1.TempDir()

	for _, name := range []string{"alpha", "beta"} {
		t.AssertNoError(os.Mkdir(filepath.Join(cwdRepos, name), 0o755))
	}
	t.AssertNoError(os.Mkdir(filepath.Join(userRepos, "gamma"), 0o755))
	// A stray non-directory entry must be skipped.
	t.AssertNoError(os.WriteFile(filepath.Join(cwdRepos, "not-a-repo"), []byte("x"), 0o644))

	p := &typeResourceProvider{
		reposDir:     cwdRepos,
		userReposDir: userRepos,
		startupIsCwd: true,
	}

	repos, err := p.scopedRepos()
	t.AssertNoError(err)

	var ids []string
	for _, r := range repos {
		ids = append(ids, r.RepoId)
	}

	got := strings.Join(ids, ",")
	expected := ".alpha,.beta,gamma"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

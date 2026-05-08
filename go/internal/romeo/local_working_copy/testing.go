//go:build test

package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

// MakeTesting builds a *Repo backed by a fresh tempdir-rooted env_repo
// for unit tests. Wraps env_repo.MakeTesting and runs the standard
// MakeWithEnvRepo pipeline so the resulting Repo has a real keypair,
// initialized config, indexes, and store. The optional contents map
// is forwarded to env_repo.MakeTesting (digest -> content) for tests
// that need pre-seeded blobs.
func MakeTesting(
	t *ui.TestContext,
	contents map[string]string,
) *Repo {
	envRepo := env_repo.MakeTesting(t, contents)
	return MakeWithEnvRepo(OptionsEmpty, envRepo)
}

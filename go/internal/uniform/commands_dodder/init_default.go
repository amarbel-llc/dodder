package commands_dodder

import (
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/piggy/go/pkgs/agent"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	bigBang := env_repo.BigBang{}
	bigBang.SetDefaults()

	utility.AddCmd("init-default", &InitDefault{
		Genesis: command_components_dodder.Genesis{
			BigBang: bigBang,
		},
	})
}

// InitDefault initializes a repository with sensible defaults for an
// unattended / per-session bootstrap: the location defaults to the
// cwd `.default` repo, the signing key is auto-detected from the SSH
// agent (a fresh per-repo key is generated when none is available), an
// existing CWD-local `.default` madder blob store is reused, and the
// zettel-id vocabulary is seeded from the embedded default word lists.
type InitDefault struct {
	command_components_dodder.Genesis
}

var (
	_ interfaces.CommandComponentWriter = (*InitDefault)(nil)
	_ command.CommandWithArgs           = (*InitDefault)(nil)
)

func (cmd *InitDefault) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "repo-id",
			Description: "location handle for the new repository (scope via spelling; defaults to the cwd .default repo)",
			Required:    false,
		}},
	}}
}

func (cmd InitDefault) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a repository with sensible defaults",
		Long: "Like `init`, but for an unattended / per-session bootstrap. " +
			"The location handle defaults to the cwd `.default` repo. The " +
			"signing key is auto-detected from the SSH agent (a fresh " +
			"per-repo key is generated when none is available), an existing " +
			"CWD-local `.default` madder blob store is reused when present, " +
			"and the zettel-id vocabulary is seeded from the embedded default " +
			"word lists. Re-running in an already-initialized directory is a no-op.",
	}
}

func (cmd *InitDefault) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.Genesis.SetFlagDefinitions(flagSet)
}

func (cmd *InitDefault) Run(req command.Request) {
	location := req.PopArgOrDefault("repo-id", "")
	req.AssertNoMoreArgs()

	cwd, err := os.Getwd()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	// The positional, when given, is the new repo's location handle (full
	// FDR-0019 grammar). When omitted, fall back to the cwd `.default` repo —
	// the per-session bootstrap default — unless -repo_id / DODDER_REPO_ID
	// already chose a location. GetConfigAny returns the shared *Config
	// pointer (OnTheFirstDay reads the same one), so the mutation sticks.
	// Same cast pattern as init-workspace. Resolve this before the
	// idempotency probe so the probe targets the repo this run will write.
	repoName := repo_id.DefaultName
	if config, ok := req.Utility.GetConfigAny().(*repo_config_cli.Config); ok {
		if location != "" {
			if err := config.RepoId.Set(location); err != nil {
				req.Cancel(err)
				return
			}
		} else if repo_id.IsAuto(config.RepoId) {
			config.RepoId = repo_id.CwdDefault()
		}
		repoName = repo_id.EffectiveName(config.RepoId)
	}

	// Idempotency: `init` is not re-runnable (it collides on the
	// inventory-lists log), so an already-initialized directory is a
	// no-op. The repo's metadata nests under repos/<name>/ (FDR-0019), so
	// probe that name. Mirrors spinclass FDR 0008's RepoReady guard.
	if pathExists(filepath.Join(
		cwd, ".dodder", "local", "share", "repos", repoName, "config-seed",
	)) {
		return
	}

	cmd.applyDefaults(cwd)

	cmd.OnTheFirstDay(req)
}

// applyDefaults wires the auto-detected signing key, blob-store reuse,
// and embedded zettel-id vocabulary onto the BigBang before genesis.
func (cmd *InitDefault) applyDefaults(cwd string) {
	// Seed the zettel-id vocabulary from the embedded default word
	// lists unless the caller supplied explicit -yin/-yang files.
	cmd.BigBang.YinDefault = true
	cmd.BigBang.YangDefault = true

	// Signing: adopt the first ed25519 key the SSH agent offers; on any
	// failure (no agent, locked agent, no key) leave PrivateKey null so
	// env_repo.Genesis generates a fresh per-repo key.
	if cmd.BigBang.PrivateKey.IsNull() {
		if key, ok := firstSSHAgentSigningKey(); ok {
			if err := cmd.BigBang.PrivateKey.Set(key); err != nil {
				cmd.BigBang.PrivateKey = markl.Id{}
			}
		} else {
			fmt.Fprintln(
				os.Stderr,
				"init-default: no SSH agent signing key found; generating a fresh per-repo key",
			)
		}
	}

	// Reuse the CWD-local `.default` madder store created by the
	// per-worktree madder pin (FDR 0003) when present, so genesis adopts
	// it instead of colliding on a second store. Only set when the
	// caller didn't pass an explicit -blob_store-id.
	if cmd.BigBang.BlobStoreId.IsEmpty() &&
		pathExists(filepath.Join(cwd, ".madder", "local", "share", "blob_stores", "default", "blob_store-config")) {
		// ".default" is a known-valid id, so the only realistic error
		// is a future parser change; ignore it and let genesis's
		// pre-flight blob-store validation surface any real problem.
		_ = cmd.BlobStore.GetFlagValueBlobIds(
			&cmd.BigBang.BlobStoreId,
		).Set(".default")
	}
}

// firstSSHAgentSigningKey returns the markl.Id text of the first
// ed25519 key the SSH agent offers, in the format `dodder init
// -private_key` accepts. ok is false on any discovery error or when no
// ed25519 key is present.
func firstSSHAgentSigningKey() (key string, ok bool) {
	keys, err := agent.DiscoverSSHAgentEd25519KeysVerbose()
	if err != nil || len(keys) == 0 {
		return "", false
	}

	text, err := keys[0].Id.MarshalText()
	if err != nil {
		return "", false
	}

	return string(text), true
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

package commands_dodder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// initDefaultRepoIdUnsafe matches characters not allowed in a repo-id
// derived from a directory name; they are collapsed to '-'.
var initDefaultRepoIdUnsafe = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

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
// unattended / per-session bootstrap: the repo-id defaults to the
// current directory name, the signing key is auto-detected from the SSH
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
			Description: "identifier for the new repository (defaults to the current directory name)",
			Required:    false,
		}},
	}}
}

func (cmd InitDefault) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a repository with sensible defaults",
		Long: "Like `init`, but for an unattended / per-session bootstrap. " +
			"The repo-id defaults to the current directory name. The signing " +
			"key is auto-detected from the SSH agent (a fresh per-repo key is " +
			"generated when none is available), an existing CWD-local " +
			"`.default` madder blob store is reused when present, and the " +
			"zettel-id vocabulary is seeded from the embedded default word " +
			"lists. Re-running in an already-initialized directory is a no-op.",
	}
}

func (cmd *InitDefault) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.Genesis.SetFlagDefinitions(flagSet)
}

func (cmd *InitDefault) Run(req command.Request) {
	repoId := req.PopArgOrDefault("repo-id", "")
	req.AssertNoMoreArgs()

	cwd, err := os.Getwd()
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	// Idempotency: `init` is not re-runnable (it collides on the
	// inventory-lists log), so an already-initialized directory is a
	// no-op. Mirrors spinclass FDR 0008's RepoReady guard.
	if pathExists(filepath.Join(cwd, ".dodder", "local", "share", "config-seed")) {
		return
	}

	if repoId == "" {
		repoId = deriveRepoIdFromDir(cwd)
	}

	// Default to a CWD-local repository (`.dodder` in the current
	// directory) — the per-session use case — unless the caller chose a
	// location via -repo_id / DODDER_REPO_ID. GetConfigAny returns the
	// shared *Config pointer (OnTheFirstDay reads the same one), so this
	// sticks. Same cast pattern as init-workspace.
	if config, ok := req.Utility.GetConfigAny().(*repo_config_cli.Config); ok {
		if config.RepoId.IsEmpty() {
			_ = config.RepoId.Set(".")
		}
	}

	cmd.applyDefaults(cwd)

	cmd.OnTheFirstDay(req, repoId)
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
	keys, err := markl.DiscoverSSHAgentEd25519KeysVerbose()
	if err != nil || len(keys) == 0 {
		return "", false
	}

	text, err := keys[0].Id.MarshalText()
	if err != nil {
		return "", false
	}

	return string(text), true
}

func deriveRepoIdFromDir(dir string) string {
	id := initDefaultRepoIdUnsafe.ReplaceAllString(filepath.Base(dir), "-")
	id = strings.Trim(id, "-")

	if id == "" {
		return "dodder-worktree"
	}

	return id
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

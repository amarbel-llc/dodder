package mcp_dodder

import (
	"context"
	"sync"
	"time"

	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/flags"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type BridgeResult struct {
	Stdout    string
	Stderr    string
	Truncated bool
	BytesSeen int
}

type Bridge struct {
	utility command.Utility

	// defaultRepoId is the server's startup repo; a tool call that omits
	// repo_id targets it. pinned reports that the server was started with
	// an explicit -repo_id, which restricts every call to that repo
	// (FDR-0019: startup pin restricts; unpinned routes per-call).
	defaultRepoId scoped_id.Id
	pinned        bool

	// mu serializes RunCommand. The go-mcp server handles every
	// incoming message on its own goroutine, but the commands the
	// bridge runs are CLI-shaped: the registry hands back long-lived
	// command values whose flag-bound state is shared across
	// invocations, so reset/parse/run must not interleave (#247).
	// Pointer so Bridge copies (it is passed by value) share the lock.
	mu *sync.Mutex
}

func MakeBridge(
	utility command.Utility,
	defaultRepoId scoped_id.Id,
	pinned bool,
) Bridge {
	return Bridge{
		utility:       utility,
		defaultRepoId: defaultRepoId,
		pinned:        pinned,
		mu:            &sync.Mutex{},
	}
}

// RunCommand runs a command against the bridge's default repo (the
// server's startup repo). Use RunCommandWithRepoId to target a specific
// repo per call.
func (b Bridge) RunCommand(
	ctx context.Context,
	cmdName string,
	cliArgs []string,
	maxBytes int,
) (BridgeResult, error) {
	return b.RunCommandWithRepoId(ctx, cmdName, cliArgs, maxBytes, "")
}

// resolveRepoId picks the repo a tool call targets: the explicit per-call
// repoIdParam (CLI spelling, e.g. "work" or ".work") when non-empty, else
// the server's startup default. A pinned server (started with an explicit
// -repo_id) rejects a per-call repo_id naming a different repo. The result
// is gated by repo_id.CheckSupported so unwired scopes (multi-dot, system)
// fail with a clear error rather than a downstream panic.
func (b Bridge) resolveRepoId(repoIdParam string) (scoped_id.Id, error) {
	effective := b.defaultRepoId

	if repoIdParam != "" {
		var id scoped_id.Id
		if err := id.Set(repoIdParam); err != nil {
			return id, errors.Wrapf(err, "invalid repo_id %q", repoIdParam)
		}

		if b.pinned && id.String() != b.defaultRepoId.String() {
			return id, errors.Errorf(
				"this MCP server is pinned to repo %q; per-call repo_id %q is not allowed",
				b.defaultRepoId.String(),
				repoIdParam,
			)
		}

		effective = id
	}

	if err := repo_id.CheckSupported(effective); err != nil {
		return effective, err
	}

	return effective, nil
}

// RunCommandWithRepoId runs a command targeting the given repo (see
// resolveRepoId). An empty repoIdParam means the server's default repo.
func (b Bridge) RunCommandWithRepoId(
	ctx context.Context,
	cmdName string,
	cliArgs []string,
	maxBytes int,
	repoIdParam string,
) (BridgeResult, error) {
	repoId, err := b.resolveRepoId(repoIdParam)
	if err != nil {
		return BridgeResult{}, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	outWriter := MakeLimitingWriter(maxBytes)
	errWriter := MakeLimitingWriter(maxBytes)

	config := repo_config_cli.Default()
	config.RepoId = repoId
	config.CustomOut = outWriter
	config.CustomErr = errWriter

	utility := command.MakeUtility("dodder", config)

	for name, cmd := range b.utility.AllCmds() {
		utility.AddCmd(name, cmd)
	}

	// The registry hands back long-lived command values shared across
	// every bridge invocation (and with the CLI registration in the
	// same process). Commands with accumulating flag-bound state
	// implement ResetCLIState so each tool call parses flags against a
	// clean slate — without this, two `new` calls concatenate
	// descriptions (#247).
	if cmd, ok := b.utility.GetCmd(cmdName); ok {
		if resetter, ok := cmd.(command.CommandWithResetCLIState); ok {
			resetter.ResetCLIState()
		}
	}

	errCtx := errors.MakeContext(ctx)

	args := make([]string, 0, 2+len(cliArgs))
	args = append(args, "dodder", cmdName)
	args = append(args, cliArgs...)

	var result BridgeResult

	if err := errCtx.Run(func(ctx errors.Context) {
		cmd, flagSet, ok := utility.MakeCmdAndFlagSet(ctx, args)
		if !ok {
			return
		}

		req, ok := utility.MakeRequest(ctx, cmd, flagSet)
		if !ok {
			return
		}

		cmd.Run(req)
	}); err != nil {
		return result, err
	}

	result = BridgeResult{
		Stdout:    outWriter.String(),
		Stderr:    errWriter.String(),
		Truncated: outWriter.Truncated(),
		BytesSeen: outWriter.BytesSeen(),
	}

	return result, nil
}

// WithRepo builds a fresh local working copy for the repo addressed by
// repoIdParam (empty = the server's default repo), gated by the same
// pin/CheckSupported resolution as the tool path (resolveRepoId), and runs
// fn against it. It is the store-backed counterpart to RunCommandWithRepoId:
// the handlers that need a real *Repo rather than a CLI run — edit,
// reset-lock, and blob-format listing (FDR-0019 #278) — call this.
//
// fn runs INSIDE the build's error context, exactly as RunCommandWithRepoId
// runs cmd.Run inside it. This is load-bearing: MakeLocalWorkingCopy
// registers the env's temp-dir cleanup (and the repo Flush) as After hooks
// on that context, so they must fire AFTER fn finishes — returning the repo
// for use after the context completed would tear its temp dirs down first,
// and a subsequent blob write would chmod into a removed directory.
//
// A fresh repo is built per call, mirroring how RunCommandWithRepoId builds
// one per command run: no held-open cache, so no lock-holding or index
// staleness. The build duration is emitted as a statsd timer so the
// build-per-call cost is observable; if it ever dominates MCP latency, switch
// to a lazy per-repo cache (the FDR "MCP repo cache" lever).
func (b Bridge) WithRepo(
	ctx context.Context,
	repoIdParam string,
	fn func(*local_working_copy.Repo) error,
) error {
	repoId, err := b.resolveRepoId(repoIdParam)
	if err != nil {
		return err
	}

	config := repo_config_cli.Default()
	config.RepoId = repoId

	utility := command.MakeUtility("dodder", config)

	errCtx := errors.MakeContext(ctx)

	return errCtx.Run(func(ctx errors.Context) {
		// MakeRequest reads the repo only from req.Utility's config
		// (config.RepoId); the flagSet is just a required, unparsed
		// placeholder and the cmd argument is unused.
		flagSet := flags.NewFlagSet("open-repo", flags.ContinueOnError)

		req, ok := utility.MakeRequest(ctx, nil, flagSet)
		if !ok {
			return
		}

		start := time.Now()
		repo := command_components_dodder.LocalWorkingCopy{}.
			MakeLocalWorkingCopy(req)
		emitTiming("dodder.mcp.open_repo", time.Since(start))

		if err := fn(repo); err != nil {
			ctx.Cancel(err)
		}
	})
}

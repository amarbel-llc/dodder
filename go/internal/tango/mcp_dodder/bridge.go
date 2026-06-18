package mcp_dodder

import (
	"context"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
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

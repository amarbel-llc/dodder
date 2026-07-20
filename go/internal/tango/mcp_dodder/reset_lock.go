package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/charlie/file_lock"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/go-mcp/protocol"
	"code.linenisgreat.com/purse-first/libs/go-mcp/server"
)

// makeResetLockHandler implements the reset-lock tool (#249): the env
// lock is intentionally left held when a mutating operation fails (see
// file_lock.ErrUnableToAcquireLock's cause text), and the interactive
// CLI recovery prompt cannot run inside the MCP server. This tool is
// the deliberate, user-approved recovery path — the clown plugin's
// PreToolUse hook forces an `ask` decision on every invocation.
func makeResetLockHandler(
	bridge Bridge,
) server.ToolHandlerV1 {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		var p struct {
			RepoId string `json:"repo_id"`
		}

		if err := json.Unmarshal(args, &p); err != nil {
			return protocol.ErrorResultV1(
				fmt.Sprintf("Invalid arguments: %v", err),
			), nil
		}

		var responseText, unsupported string

		// Open the addressed repo per call (FDR-0019 #278) and break its lock
		// within the repo's context lifecycle; empty repo_id resolves to the
		// server's default.
		if err := bridge.WithRepo(
			ctx,
			p.RepoId,
			func(repo *local_working_copy.Repo) error {
				lockSmith := repo.GetEnvRepo().GetLockSmith()

				breaker, ok := lockSmith.(file_lock.Breaker)
				if !ok {
					unsupported = fmt.Sprintf(
						"this repo's lock (%T) does not support forced resets",
						lockSmith,
					)
					return nil
				}

				result, err := breaker.Break()
				if err != nil {
					return err
				}

				responseText = resetLockResponseText(result, breaker.Path())
				return nil
			},
		); err != nil {
			return protocol.ErrorResultV1(formatToolError(err)), nil
		}

		if unsupported != "" {
			return protocol.ErrorResultV1(unsupported), nil
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(responseText),
			},
		}, nil
	}
}

// resetLockResponseText states the outcome of a lock reset
// unambiguously: what was found, what was cleared, and whether the
// repo is locked NOW. Callers must never have to infer the resulting
// state (#249).
func resetLockResponseText(
	result file_lock.BreakResult,
	path string,
) string {
	switch {
	case result.ReleasedHandle && result.RemovedFile:
		return fmt.Sprintf(
			"the repo is no longer locked: released this server's in-process lock handle and removed the lock file at %q",
			path,
		)

	case result.RemovedFile:
		return fmt.Sprintf(
			"the repo is no longer locked: removed the stale lock file at %q (no in-process handle was held)",
			path,
		)

	case result.ReleasedHandle:
		return fmt.Sprintf(
			"the repo is no longer locked: released this server's in-process lock handle; no lock file existed at %q",
			path,
		)

	default:
		return fmt.Sprintf(
			"the repo was not locked: no in-process handle was held and no lock file existed at %q; nothing was reset",
			path,
		)
	}
}

// formatToolError prefixes lock-classified failures with an
// unambiguous statement of the repo's lock state and the recovery
// path, falling through to formatErrorDetail for everything else. The
// env lock being intentionally left held after a failed mutation is
// invisible in a bare error tree — and in the long-lived MCP server it
// poisons every subsequent mutating call (#249).
func formatToolError(err error) string {
	var inProcess file_lock.ErrAlreadyLockedInProcess
	if errors.As(err, &inProcess) {
		return fmt.Sprintf(
			"REPO LOCKED: the environment lock at %q is held by this MCP server process — a previous mutating operation failed and intentionally left it held for recovery. Every mutating tool call will fail until the lock is reset. Recovery: call the reset-lock tool (requires user approval).\n\n%s",
			inProcess.Path,
			formatErrorDetail(err),
		)
	}

	var onDisk file_lock.ErrUnableToAcquireLock
	if errors.As(err, &onDisk) {
		return fmt.Sprintf(
			"REPO LOCKED: a lock file exists at %q — either a previous failed operation left it behind, or another process holds it right now. Verify no other dodder process is using this repo, then call the reset-lock tool (requires user approval) to recover.\n\n%s",
			onDisk.Path,
			formatErrorDetail(err),
		)
	}

	// The shape lock failures actually take through the bridge: the
	// retryable ErrUnableToAcquireLock enters Recover, the no-TTY
	// Confirm declines, and Recover aborts with this non-retryable
	// carrier (see file_lock's recovery-loop comment for why it does
	// not wrap the retryable).
	var recoveryAborted file_lock.ErrLockRecoveryAborted
	if errors.As(err, &recoveryAborted) {
		return fmt.Sprintf(
			"REPO LOCKED: a lock file exists at %q and interactive recovery was declined or unavailable — either a previous failed operation left it behind, or another process holds it right now. Verify no other dodder process is using this repo, then call the reset-lock tool (requires user approval) to recover.\n\n%s",
			recoveryAborted.Path,
			formatErrorDetail(err),
		)
	}

	return formatErrorDetail(err)
}

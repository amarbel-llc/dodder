package file_lock

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	pkgErrDisamb struct{}
	pkgError     = errors.Typed[pkgErrDisamb]
)

type ErrLockRequired struct {
	Operation string
}

func (err ErrLockRequired) Is(target error) bool {
	_, ok := target.(ErrLockRequired)
	return ok
}

func (err ErrLockRequired) Error() string {
	return fmt.Sprintf(
		"lock required for operation: %q",
		err.Operation,
	)
}

func (err ErrLockRequired) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

// ErrAlreadyLockedInProcess reports a Lock() attempt while this
// process already holds the lock (lock.file != nil). Distinct from
// ErrUnableToAcquireLock (another process's lock file on disk): this
// branch never touches the filesystem and previously surfaced as a
// bare stack-frame error, which made the long-lived MCP server's
// stuck-lock state impossible to classify (#249).
type ErrAlreadyLockedInProcess struct {
	Path string
}

func (err ErrAlreadyLockedInProcess) Error() string {
	return "already locked by this process"
}

func (err ErrAlreadyLockedInProcess) Is(target error) bool {
	_, ok := target.(ErrAlreadyLockedInProcess)
	return ok
}

func (err ErrAlreadyLockedInProcess) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

type ErrUnableToAcquireLock struct {
	envUI       env_ui.Env
	Path        string
	description string
}

var _ interfaces.ErrorRetryable = ErrUnableToAcquireLock{}

func (err ErrUnableToAcquireLock) Error() string {
	return fmt.Sprintf("%s is currently locked", err.description)
}

func (err ErrUnableToAcquireLock) Is(target error) bool {
	_, ok := target.(ErrUnableToAcquireLock)
	return ok
}

func (err ErrUnableToAcquireLock) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

func (err ErrUnableToAcquireLock) GetErrorCause() []string {
	return []string{
		fmt.Sprintf(
			"A previous operation that acquired the %s lock failed.",
			err.description,
		),
		"The lock is intentionally left behind in case recovery is necessary.",
	}
}

func (err ErrUnableToAcquireLock) GetErrorRecovery() []string {
	return []string{
		fmt.Sprintf("The lockfile needs to removed (`rm %q`).", err.Path),
	}
}

func (err ErrUnableToAcquireLock) Recover(
	ctx interfaces.ActiveContext,
	retry interfaces.FuncRetry,
	abort interfaces.FuncRetryAborted,
) {
	errors.PrintHelpful(err.envUI.GetErr(), err)

	if err.envUI.Confirm("delete the existing lock?", "") {
		if err := os.Remove(err.Path); err != nil {
			ctx.Cancel(err)
		}

		retry()
	} else {
		// Abort with a NON-retryable typed error that carries the lock
		// path but does NOT wrap the retryable receiver. Wrapping the
		// receiver here loops forever: abort routes through
		// ctx.Cancel(errContextRetryAborted{underlying: ...}), dewey's
		// cancel() runs errors.As for ErrorRetryable on that error, and
		// with errContextRetryAborted unwrapping its underlying
		// (purse-first#145) it finds THIS error again and re-enters
		// Recover — observed as `dodder mcp` spinning at 100% CPU
		// (#249). Callers classify lock failures via errors.As on
		// ErrLockRecoveryAborted instead. Once dewey's cancel() treats
		// errContextRetryAborted as terminal (purse-first#147), this
		// can revert to wrapping the receiver.
		abort(ErrLockRecoveryAborted{Path: err.Path})
	}
}

// ErrLockRecoveryAborted reports a declined (or undeliverable, e.g.
// no-TTY) interactive lock recovery. Deliberately NOT ErrorRetryable
// and deliberately NOT wrapping ErrUnableToAcquireLock — see the abort
// comment in Recover for the recovery-loop hazard.
type ErrLockRecoveryAborted struct {
	Path string
}

func (err ErrLockRecoveryAborted) Error() string {
	return fmt.Sprintf(
		"not deleting the lock at %q; the repo stays locked",
		err.Path,
	)
}

func (err ErrLockRecoveryAborted) Is(target error) bool {
	_, ok := target.(ErrLockRecoveryAborted)
	return ok
}

func (err ErrLockRecoveryAborted) GetErrorType() pkgErrDisamb {
	return pkgErrDisamb{}
}

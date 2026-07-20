package file_lock

import (
	"io/fs"
	"os"
	"sync"
	"time"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/files"
)

type Lock struct {
	envUI       env_ui.Env
	path        string
	description string
	mutex       sync.Mutex
	file        *os.File
}

// TODO switch to using context
func New(
	envUI env_ui.Env,
	path string,
	description string,
) (l *Lock) {
	return &Lock{
		envUI:       envUI,
		path:        path,
		description: description,
	}
}

func (lock *Lock) Path() string {
	return lock.path
}

func (lock *Lock) IsAcquired() (acquired bool) {
	lock.mutex.Lock()
	defer lock.mutex.Unlock()

	acquired = lock.file != nil

	return acquired
}

func (lock *Lock) Lock() (err error) {
	if !lock.mutex.TryLock() {
		err = errors.ErrorWithStackf("attempting concurrent locks")
		return err
	}

	defer lock.mutex.Unlock()

	if lock.file != nil {
		err = ErrAlreadyLockedInProcess{Path: lock.Path()}
		return err
	}

	createLock := func(path string) (*os.File, error) {
		return files.TryOrTimeout(
			path,
			time.Second,
			func(path string) (*os.File, error) {
				return files.OpenFile(
					path,
					os.O_RDONLY|os.O_EXCL|os.O_CREATE,
					0o755,
				)
			},
			"acquiring lock",
		)
	}

	if lock.file, err = files.TryOrMakeDirIfNecessary(
		lock.Path(),
		createLock,
	); err != nil {
		if errors.Is(err, fs.ErrExist) {
			err = ErrUnableToAcquireLock{
				envUI:       lock.envUI,
				Path:        lock.Path(),
				description: lock.description,
			}
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	return err
}

// BreakResult reports what Break actually found and cleared, so
// callers (the MCP reset-lock tool) can state the outcome
// unambiguously instead of guessing.
type BreakResult struct {
	// ReleasedHandle: this process held the lock and the in-memory
	// handle was closed and cleared.
	ReleasedHandle bool
	// RemovedFile: a lock file existed on disk and was removed.
	RemovedFile bool
}

// Breaker is the optional recovery surface on a LockSmith. The
// interfaces.LockSmith contract (dewey) stays Lock/Unlock/IsAcquired;
// recovery callers type-assert to this.
type Breaker interface {
	Break() (BreakResult, error)
	Path() string
}

// Break forcibly resets the lock: it closes and clears an in-process
// handle if one is held, and removes the lock file if one exists.
// Unlike Unlock it performs no flushes and tolerates every absent
// state — it is the recovery path for a lock that a failed operation
// intentionally left behind (see ErrUnableToAcquireLock's cause text),
// reachable in contexts where the interactive Recover prompt cannot
// run (#249).
func (lock *Lock) Break() (result BreakResult, err error) {
	if !lock.mutex.TryLock() {
		err = errors.ErrorWithStackf("attempting concurrent locks")
		return result, err
	}

	defer lock.mutex.Unlock()

	if lock.file != nil {
		// Close errors are not actionable during a forced reset; the
		// file is being removed regardless.
		_ = lock.file.Close()
		lock.file = nil
		result.ReleasedHandle = true
	}

	switch removeErr := os.Remove(lock.Path()); {
	case removeErr == nil:
		result.RemovedFile = true
	case errors.Is(removeErr, fs.ErrNotExist):
		// nothing on disk; fine
	default:
		err = errors.Wrap(removeErr)
		return result, err
	}

	return result, err
}

func (lock *Lock) Unlock() (err error) {
	if !lock.mutex.TryLock() {
		err = errors.ErrorWithStackf("attempting concurrent locks")
		return err
	}

	defer lock.mutex.Unlock()

	if err = lock.file.Close(); err != nil {
		err = errors.Wrapf(err, "File: %v", lock.file)
		return err
	}

	lock.file = nil

	if err = os.Remove(lock.Path()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

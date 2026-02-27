package file_lock

import (
	"io/fs"
	"os"
	"sync"
	"time"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
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
		err = errors.ErrorWithStackf("already locked")
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

package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

// TODO move into own package

type MoveOptions struct {
	TemporaryFS
	ErrorOnAttemptedOverwrite   bool
	FinalPathOrDir              string
	GenerateFinalPathFromDigest bool
}

type localFileMover struct {
	funcJoin func(string, ...string) string
	file     *os.File
	domain_interfaces.BlobWriter

	basePath                  string
	blobPath                  string
	lockFile                  bool
	errorOnAttemptedOverwrite bool
}

func NewMover(
	config Config,
	moveOptions MoveOptions,
) (domain_interfaces.BlobWriter, error) {
	// TODO make MoveOptions an interface and add support for localFileShaMover
	// and localFinalPathMover
	return newMover(config, moveOptions)
}

// TODO add back support for locking internal files
// TODO split mover into sha-based mover and final-path based mover
// TODO extract writer portion in injected depenency
func newMover(
	config Config,
	moveOptions MoveOptions,
) (mover *localFileMover, err error) {
	mover = &localFileMover{
		funcJoin:                  config.funcJoin,
		errorOnAttemptedOverwrite: moveOptions.ErrorOnAttemptedOverwrite,
	}

	if moveOptions.GenerateFinalPathFromDigest {
		mover.basePath = moveOptions.FinalPathOrDir

		if mover.basePath == "" {
			err = errors.ErrorWithStackf("basepath is nil")
			return mover, err
		}
	} else {
		mover.blobPath = moveOptions.FinalPathOrDir
	}

	if mover.file, err = moveOptions.FileTemp(); err != nil {
		err = errors.Wrap(err)
		return mover, err
	}

	if mover.BlobWriter, err = NewWriter(
		config,
		mover.file,
	); err != nil {
		err = errors.Wrap(err)
		return mover, err
	}

	return mover, err
}

func (mover *localFileMover) Close() (err error) {
	if mover.file == nil {
		err = errors.ErrorWithStackf("nil file")
		return err
	}

	if mover.BlobWriter == nil {
		err = errors.ErrorWithStackf("nil object reader")
		return err
	}

	if err = mover.BlobWriter.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// var fi os.FileInfo

	// if fi, err = m.file.Stat(); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	if err = mover.file.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	digest := mover.GetMarklId()

	if err = markl.MakeErrEmptyType(digest); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if digest.IsNull() {
		return err
	}

	// if err = merkle.MakeErrIsNull(digest, ""); err != nil {
	// 	err = errors.Wrap(err)
	// 	return
	// }

	// log.Log().Printf(
	// 	"wrote %d bytes to %s, sha %s",
	// 	fi.Size(),
	// 	m.file.Name(),
	// 	sh,
	// )

	if mover.blobPath == "" {
		// TODO-P3 move this validation to options
		if mover.blobPath, err = MakeDirIfNecessary(
			markl.FormatBytesAsHex(digest),
			mover.funcJoin,
			mover.basePath,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	path := mover.file.Name()

	if err = os.Rename(path, mover.blobPath); err != nil {
		if files.Exists(mover.blobPath) {
			if mover.errorOnAttemptedOverwrite {
				err = MakeErrBlobAlreadyExists(digest, mover.blobPath)
			} else {
				err = nil
			}

			return err
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	// log.Log().Printf("moved %s to %s", p, m.objectPath)

	if mover.lockFile {
		if err = files.SetDisallowUserChanges(mover.blobPath); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

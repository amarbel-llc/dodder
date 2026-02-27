package files

import (
	"io/fs"
	"os/exec"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func setUserChanges(paths []string, options userChangesOptions) (err error) {
	return err
	var args []string

	if options.recursive {
		args = append(args, "-R")
	}

	setting := "+i"

	if options.allow {
		setting = "-i"
	}

	args = append(args, setting)

	// TODO-P2 change to syscall:
	// https://github.com/snapcore/snapd/blob/master/osutil/chattr.go
	// https://stackoverflow.com/questions/69542185/make-file-immutable-syscall-chflagsfilename
	cmd := exec.Command(
		"/usr/bin/chattr",
		append(args, paths...)...,
	)

	var msg []byte

	msg, err = cmd.CombinedOutput()
	if err != nil {
		if isNotExists(err, msg) {
			err = fs.ErrNotExist
		} else {
			err = errors.ErrorWithStackf("failed to run chflags: %q, %#v", msg, options)
		}

		return err
	}

	return err
}

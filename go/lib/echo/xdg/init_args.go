package xdg

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type InitArgs struct {
	Home        string
	Cwd         string
	UtilityName string
	ExecPath    string
	Pid         int

	OverrideEnvVarName string
}

func (initArgs *InitArgs) Initialize(utilityName string) (err error) {
	if initArgs.Home == "" {
		if initArgs.Home, err = os.UserHomeDir(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if initArgs.Cwd == "" {
		if initArgs.Cwd, err = os.Getwd(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if initArgs.OverrideEnvVarName != "" {
		if utilityNameOverride := os.Getenv(initArgs.OverrideEnvVarName); utilityNameOverride != "" {
			utilityName = utilityNameOverride
		}
	}

	if initArgs.ExecPath, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	initArgs.Pid = os.Getpid()

	initArgs.UtilityName = utilityName

	return err
}

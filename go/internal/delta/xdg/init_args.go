package xdg

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

const EnvXDGUtilityNameOverride = "DODDER_XDG_UTILITY_OVERRIDE"

type InitArgs struct {
	Home        string
	Cwd         string
	UtilityName string
	ExecPath    string
	Pid         int
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

	// TODO accept EnvVarName from args instead of hardcoding
	if utilityNameOverride := os.Getenv(EnvXDGUtilityNameOverride); utilityNameOverride != "" {
		utilityName = utilityNameOverride
	}

	if initArgs.ExecPath, err = os.Executable(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	initArgs.Pid = os.Getpid()

	initArgs.UtilityName = utilityName

	return err
}

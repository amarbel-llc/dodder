package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/debug"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
)

type beforeXDG struct {
	xdgInitArgs xdg.InitArgs

	dryRun       bool
	debugOptions debug.Options
}

func (env *beforeXDG) initialize(
	debugOptions debug.Options,
	utilityName string,
) (err error) {
	env.debugOptions = debugOptions
	env.xdgInitArgs.OverrideEnvVarName = "DODDER_XDG_UTILITY_OVERRIDE"

	if err = env.xdgInitArgs.Initialize(utilityName); err != nil {
		err = errors.Wrap(err)
		return err
	}

	env.dryRun = debugOptions.DryRun

	// TODO switch to useing MakeCommonEnv()
	{
		if err = os.Setenv(EnvBin, env.xdgInitArgs.ExecPath); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

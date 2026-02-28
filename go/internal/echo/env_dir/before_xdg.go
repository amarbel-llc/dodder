package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/debug"
	"code.linenisgreat.com/dodder/go/lib/echo/xdg"
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

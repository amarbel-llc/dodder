package env_dir

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/echo/xdg"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/debug"
)

// TODO separate read-only from write

func MakeDefault(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
) env {
	return MakeWithDefaultHome(
		context,
		utilityName,
		debugOptions,
		true,
		true,
	)
}

func MakeDefaultNoInit(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
) env {
	return MakeWithDefaultHome(
		context,
		utilityName,
		debugOptions,
		true,
		false,
	)
}

func MakeFromXDGDotenvPath(
	context errors.Context,
	debugOptions debug.Options,
	xdgDotenvPath string,
) env {
	dotenv := xdg.Dotenv{
		XDG: &xdg.XDG{},
	}

	var file *os.File

	{
		var err error

		if file, err = os.Open(xdgDotenvPath); err != nil {
			context.Cancel(err)
		}
	}

	if _, err := dotenv.ReadFrom(file); err != nil {
		context.Cancel(err)
	}

	if err := file.Close(); err != nil {
		context.Cancel(err)
	}

	return MakeWithXDG(
		context,
		debugOptions,
		*dotenv.XDG,
	)
}

func MakeDefaultAndInitialize(
	context errors.Context,
	utilityName string,
	do debug.Options,
	overrideXDGWithCwd bool,
) env {
	var home string

	{
		var err error

		if home, err = os.UserHomeDir(); err != nil {
			context.Cancel(err)
		}
	}

	if overrideXDGWithCwd {
		var cwd string

		{
			var err error

			if cwd, err = os.Getwd(); err != nil {
				context.Cancel(err)
			}
		}

		return MakeWithXDGRootOverrideHomeAndInitialize(
			context,
			cwd,
			utilityName,
			do,
		)
	} else {
		return MakeWithHomeAndInitialize(
			context,
			utilityName,
			home,
			do,
		)
	}
}

func MakeWithDefaultHome(
	context errors.Context,
	utilityName string,
	debugOptions debug.Options,
	permitCwdXDGOverride bool,
	initialize bool,
) (env env) {
	env.Context = context

	if err := env.beforeXDG.initialize(debugOptions, utilityName); err != nil {
		env.Cancel(err)
		return env
	}

	if !initialize {
		return env
	}

	if permitCwdXDGOverride {
		if err := env.XDG.InitializeOverriddenIfNecessary(env.xdgInitArgs); err != nil {
			env.Cancel(err)
			return env
		}
	} else {
		if err := env.XDG.InitializeStandardFromEnv(env.xdgInitArgs); err != nil {
			env.Cancel(err)
			return env
		}
	}

	if err := env.initializeXDG(); err != nil {
		env.Cancel(err)
		return env
	}

	env.After(env.resetTempOnExit)

	return env
}

func MakeWithXDGRootOverrideHomeAndInitialize(
	context errors.Context,
	xdgRootOverride string,
	utilityName string,
	debugOptions debug.Options,
) (env env) {
	env.Context = context
	env.xdgInitArgs.Cwd = xdgRootOverride

	if err := env.beforeXDG.initialize(debugOptions, utilityName); err != nil {
		env.Cancel(err)
		return env
	}

	if err := env.XDG.InitializeOverridden(
		env.xdgInitArgs,
		xdgRootOverride,
	); err != nil {
		env.Cancel(err)
		return env
	}

	if err := env.initializeXDG(); err != nil {
		env.Cancel(err)
		return env
	}

	env.After(env.resetTempOnExit)

	return env
}

func MakeWithHomeAndInitialize(
	context errors.Context,
	utilityName string,
	home string,
	debugOptions debug.Options,
) (env env) {
	env.Context = context

	if err := env.beforeXDG.initialize(debugOptions, utilityName); err != nil {
		env.Cancel(err)
	}

	if err := env.XDG.InitializeStandardFromEnv(env.xdgInitArgs); err != nil {
		env.Cancel(err)
		return env
	}

	if err := env.initializeXDG(); err != nil {
		env.Cancel(err)
		return env
	}

	env.After(env.resetTempOnExit)

	return env
}

func MakeWithXDG(
	context errors.Context,
	debugOptions debug.Options,
	xdg xdg.XDG,
) (env env) {
	env.Context = context
	env.XDG = xdg

	if err := env.beforeXDG.initialize(debugOptions, xdg.UtilityName); err != nil {
		env.Cancel(err)
		return env
	}

	if err := env.initializeXDG(); err != nil {
		env.Cancel(err)
		return env
	}

	return env
}

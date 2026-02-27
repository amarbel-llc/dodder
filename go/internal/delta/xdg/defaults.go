package xdg

import "code.linenisgreat.com/dodder/go/lib/bravo/env_vars"

type DefaultEnvVar struct {
	Name                 string
	DefaultValueTemplate string
	overridden           string
}

var (
	DefaultHome = DefaultEnvVar{
		Name:                 "HOME",
		DefaultValueTemplate: "$HOME",
		overridden:           "$HOME",
	}

	DefaultData = DefaultEnvVar{
		Name:                 "XDG_DATA_HOME",
		DefaultValueTemplate: "$HOME/.local/share",
		overridden:           "$HOME/local/share",
	}

	DefaultConfig = DefaultEnvVar{
		Name:                 "XDG_CONFIG_HOME",
		DefaultValueTemplate: "$HOME/.config",
		overridden:           "$HOME/config",
	}

	DefaultState = DefaultEnvVar{
		Name:                 "XDG_STATE_HOME",
		DefaultValueTemplate: "$HOME/.local/state",
		overridden:           "$HOME/local/state",
	}

	DefaultCache = DefaultEnvVar{
		Name:                 "XDG_CACHE_HOME",
		DefaultValueTemplate: "$HOME/.cache",
		overridden:           "$HOME/cache",
	}

	DefaultRuntime = DefaultEnvVar{
		Name:                 "XDG_RUNTIME_HOME",
		DefaultValueTemplate: "$HOME/.local/runtime",
		overridden:           "$HOME/local/runtime",
	}
)

func (defaultEnvVar DefaultEnvVar) MakeBaseEnvVar(
	actual string,
) env_vars.DirectoryLayoutBaseEnvVar {
	return env_vars.DirectoryLayoutBaseEnvVar{
		Name:                 defaultEnvVar.Name,
		DefaultValueTemplate: defaultEnvVar.DefaultValueTemplate,
		ActualValue:          actual,
	}
}

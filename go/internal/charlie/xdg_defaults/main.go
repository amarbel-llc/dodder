package xdg_defaults

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_vars"
)

const (
	VarXDGOverride = "XDG_OVERRIDE"
	VarUtilityName = "XDG_UTILITY_NAME"
)

type DefaultEnvVar struct {
	Name             string
	TemplateDefault  string
	TemplateOverride string
}

var (
	Home = DefaultEnvVar{
		Name:             "HOME",
		TemplateDefault:  "$HOME",
		TemplateOverride: "$HOME",
	}

	Cwd = DefaultEnvVar{
		Name:             "PWD",
		TemplateDefault:  "$PWD",
		TemplateOverride: "$PWD",
	}

	Data = DefaultEnvVar{
		Name:             "XDG_DATA_HOME",
		TemplateDefault:  "$HOME/.local/share/$XDG_UTILITY_NAME",
		TemplateOverride: "$XDG_OVERRIDE/.$XDG_UTILITY_NAME/local/share",
	}

	Config = DefaultEnvVar{
		Name:             "XDG_CONFIG_HOME",
		TemplateDefault:  "$HOME/.config/$XDG_UTILITY_NAME",
		TemplateOverride: "$XDG_OVERRIDE/.$XDG_UTILITY_NAME/config",
	}

	State = DefaultEnvVar{
		Name:             "XDG_STATE_HOME",
		TemplateDefault:  "$HOME/.local/state/$XDG_UTILITY_NAME",
		TemplateOverride: "$XDG_OVERRIDE/.$XDG_UTILITY_NAME/local/state",
	}

	Cache = DefaultEnvVar{
		Name:             "XDG_CACHE_HOME",
		TemplateDefault:  "$HOME/.cache/$XDG_UTILITY_NAME",
		TemplateOverride: "$XDG_OVERRIDE/.$XDG_UTILITY_NAME/cache",
	}

	Runtime = DefaultEnvVar{
		Name:             "XDG_RUNTIME_HOME",
		TemplateDefault:  "$HOME/.local/runtime/$XDG_UTILITY_NAME",
		TemplateOverride: "$XDG_OVERRIDE/.$XDG_UTILITY_NAME/local/runtime",
	}
)

func (defaultEnvVar DefaultEnvVar) MakeBaseEnvVar(
	actual string,
) env_vars.DirectoryLayoutBaseEnvVar {
	return env_vars.DirectoryLayoutBaseEnvVar{
		Name:                 defaultEnvVar.Name,
		DefaultValueTemplate: defaultEnvVar.TemplateDefault,
		ActualValue:          actual,
	}
}

func MakeGetenv(
	getenv env_vars.Getenv,
	xdgOverride string,
	utilityName string,
) env_vars.Getenv {
	return func(envVarName string) string {
		var envVarValue string

		switch envVarName {
		case VarXDGOverride:
			envVarValue = xdgOverride

		case VarUtilityName:
			envVarValue = utilityName

		default:
			envVarValue = getenv(envVarName)

		}

		return envVarValue
	}
}

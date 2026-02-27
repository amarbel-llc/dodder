package cli

import (
	"maps"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

// TODO add support for comma-separated values
type CLICompleter interface {
	GetCLICompletion() map[string]string
}

type FlagValueWithCompetion interface {
	interfaces.FlagValue
	CLICompleter
}

func FlagSetVarWithCompletion(
	flagSet interfaces.CLIFlagDefinitions, value FlagValueWithCompetion,
	key string,
) {
	flagSet.Var(
		value,
		key,
		strings.Join(
			slices.Collect(
				maps.Keys(value.GetCLICompletion()),
			),
			", ",
		),
	)
}

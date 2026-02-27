package flags

import "code.linenisgreat.com/dodder/go/internal/_/interfaces"

type Definitions interface {
	BoolVar(variable *bool, name string, defaultValue bool, usage string)
	StringVar(variable *string, name string, defaultValue string, usage string)
	Var(value interfaces.FlagValue, name string, usage string)
}

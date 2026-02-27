package checkout_options

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
)

type Options struct {
	CheckoutMode checkout_mode.Mode
	OptionsWithoutMode
}

type OptionsWithoutMode struct {
	Force                bool
	AllowConflicted      bool
	StoreSpecificOptions any
}

var _ interfaces.CommandComponentWriter = (*Options)(nil)

func (c *Options) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(&c.CheckoutMode, "mode", "mode for checking out the zettel")
	flagSet.BoolVar(
		&c.Force,
		"force",
		false,
		"force update checked out zettels, even if they will overwrite existing checkouts",
	)
}

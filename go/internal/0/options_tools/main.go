package options_tools

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/google/shlex"
)

//go:generate tommy generate
type Options struct {
	Merge []string `toml:"merge"`
}

var _ interfaces.CommandComponentWriter = (*Options)(nil)

func (options *Options) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	flagSet.Func(
		"merge-tool",
		"utility to launch for merge conflict resolution",
		func(value string) (err error) {
			if options.Merge, err = shlex.Split(value); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)
}

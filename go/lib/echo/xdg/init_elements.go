package xdg

import (
	"code.linenisgreat.com/dodder/go/lib/charlie/env_vars"
	"code.linenisgreat.com/dodder/go/lib/delta/xdg_defaults"
)

type initElement struct {
	defawlt xdg_defaults.DefaultEnvVar
	actual  *env_vars.DirectoryLayoutBaseEnvVar
}

func (xdg *XDG) getInitElements() []initElement {
	return []initElement{
		{
			defawlt: xdg_defaults.Data,
			actual:  &xdg.Data,
		},
		{
			defawlt: xdg_defaults.Config,
			actual:  &xdg.Config,
		},
		{
			defawlt: xdg_defaults.State,
			actual:  &xdg.State,
		},
		{
			defawlt: xdg_defaults.Cache,
			actual:  &xdg.Cache,
		},
		{
			defawlt: xdg_defaults.Runtime,
			actual:  &xdg.Runtime,
		},
	}
}

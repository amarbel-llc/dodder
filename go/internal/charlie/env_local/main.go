package env_local

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
)

type (
	ui  = env_ui.Env
	dir = env_dir.Env
)

type Env interface {
	ui
	dir
}

type env struct {
	ui
	dir
}

func Make(ui ui, dir dir) env {
	return env{
		ui:  ui,
		dir: dir,
	}
}

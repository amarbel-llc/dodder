package env_ui

import "io"

type OptionsGetter interface {
	GetEnvOptions() Options
}

type Options struct {
	UIFileIsStderr   bool
	IgnoreTtyState   bool
	UIPrintingPrefix string
	CustomOut        io.Writer
	CustomErr        io.Writer
}

// Package env_local is a thin forwarder to madder/pkgs/env_local.
//
// Pre-#151 this package owned a dodder-flavored env_local that
// composed dodder env_ui (with FormatColor / FormatOutput /
// StringFormatWriter extras) + dodder env_dir. Stage B's env_dir
// fork drop aliased dodder env_dir.Env to madder's, and the
// env_ui extras moved to package-level functions in env_ui (so
// dodder env_ui.Env structurally matches madder env_ui.Env).
// What's left is exactly madder's env_local — keeping the import
// path stable for callers via type alias and Make forwarder.
package env_local

import (
	mad_env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
)

// Env is aliased to madder's env_local.Env. Underlying interface
// is `interface { madder env_ui.Env; madder env_dir.Env }`. dodder
// env_ui values structurally satisfy madder env_ui.Env (same
// method set after the format-components extraction); dodder
// env_dir is itself a forwarder.
type Env = mad_env_local.Env

// Make forwards to madder's. Accepts dodder env_ui.Env / env_dir.Env
// values via structural interface satisfaction.
var Make = mad_env_local.Make

package env_ui

import mad_env_ui "github.com/amarbel-llc/madder/go/pkgs/env_ui"

type OptionsGetter interface {
	GetEnvOptions() Options
}

// Options is aliased to madder's env_ui.Options so dodder env_ui.Env
// values structurally satisfy madder's env_ui.Env interface (whose
// GetOptions returns madder's Options; without the alias the
// return-type mismatch breaks satisfaction even though the structs
// are byte-identical). Required for #151 bucket B Stage A — env_repo
// hands a dodder env_local through to madder's
// blob_store_env.MakeBlobStoreEnv, which expects madder's env_local.
type Options = mad_env_ui.Options

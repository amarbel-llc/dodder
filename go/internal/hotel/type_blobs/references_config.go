package type_blobs

import "code.linenisgreat.com/dodder/go/lib/delta/script_config"

type ReferencesConfig struct {
	script_config.ScriptConfig
	Optional bool `toml:"optional,omitempty"`
}

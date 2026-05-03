package type_blobs

import "code.linenisgreat.com/dodder/go/lib/bravo/script_config"

//go:generate tommy generate
type ReferencesConfig struct {
	script_config.ScriptConfig
	Optional bool `toml:"optional,omitempty"`
}

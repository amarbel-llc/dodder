package local_working_copy

import _ "embed"

//go:embed embedded/pandoc/filters/dodder-common.lua
var embeddedPandocCommonFilter []byte

//go:embed embedded/pandoc/filters/dodder-edit.lua
var embeddedPandocEditFilter []byte

//go:embed embedded/pandoc/defaults/dodder-edit.yaml
var embeddedPandocEditDefaults []byte

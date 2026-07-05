package local_working_copy

import _ "embed"

//go:embed embedded/pandoc/filters/dodder-common.lua
var embeddedPandocCommonFilter []byte

//go:embed embedded/pandoc/filters/dodder-edit.lua
var embeddedPandocEditFilter []byte

//go:embed embedded/pandoc/filters/dodder-render.lua
var embeddedPandocRenderFilter []byte

//go:embed embedded/pandoc/defaults/dodder-edit.yaml
var embeddedPandocEditDefaults []byte

//go:embed embedded/pandoc/defaults/dodder-render.yaml
var embeddedPandocRenderDefaults []byte

//go:embed embedded/pandoc/defaults/dodder-html-partial.yaml
var embeddedPandocHtmlPartialDefaults []byte

//go:embed embedded/pandoc/defaults/dodder-html.yaml
var embeddedPandocHtmlDefaults []byte

//go:embed embedded/pandoc/defaults/dodder-gdoc.yaml
var embeddedPandocGdocDefaults []byte

//go:embed embedded/pandoc/defaults/dodder-beamer.yaml
var embeddedPandocBeamerDefaults []byte

//go:embed embedded/actionable/actionable-common.lua
var embeddedActionableCommon []byte

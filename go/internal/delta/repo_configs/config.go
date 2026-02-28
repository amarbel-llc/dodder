package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/options_print"
	"code.linenisgreat.com/dodder/go/internal/_/options_tools"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
)

type Config struct {
	DefaultType    ids.Type
	DefaultTags    ids.TagSet
	FileExtensions file_extensions.Config
	PrintOptions   options_print.Overlay
	ToolOptions    options_tools.Options
}

func MakeConfigFromOverlays(base Config, overlays ...ConfigOverlay) Config {
	return Config{}
}

func (config Config) GetToolOptions() options_tools.Options {
	return config.ToolOptions
}

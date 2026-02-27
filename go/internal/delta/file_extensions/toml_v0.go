package file_extensions

import "code.linenisgreat.com/dodder/go/internal/alfa/equals"

type TOMLV0 struct {
	Organize *string `toml:"organize"`
	Repo     *string `toml:"kasten"`
	Tag      *string `toml:"etikett"`
	Type     *string `toml:"typ"`
	Zettel   *string `toml:"zettel"`
}

func (config TOMLV0) GetFileExtensionsOverlay() Overlay {
	return Overlay{
		Organize: config.Organize,
		Repo:     config.Repo,
		Tag:      config.Tag,
		Type:     config.Type,
		Zettel:   config.Zettel,
	}
}

func (config *TOMLV0) Reset() {
	equals.SetIfNotNil(config.Organize, "")
	equals.SetIfNotNil(config.Repo, "")
	equals.SetIfNotNil(config.Tag, "")
	equals.SetIfNotNil(config.Type, "")
	equals.SetIfNotNil(config.Zettel, "")
}

func (config *TOMLV0) ResetWith(b TOMLV0) {
	config.Organize = b.Organize
	config.Repo = b.Repo
	config.Tag = b.Tag
	config.Type = b.Type
	config.Zettel = b.Zettel
}

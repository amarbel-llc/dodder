package file_extensions

import "code.linenisgreat.com/dodder/go/internal/alfa/equals"

type TOMLV1 struct {
	Config   *string `toml:"config"`
	Conflict *string `toml:"conflict"`
	Lockfile *string `toml:"lockfile"`
	Organize *string `toml:"organize"`
	Repo     *string `toml:"repo"`
	Tag      *string `toml:"tag"`
	Type     *string `toml:"type"`
	Zettel   *string `toml:"zettel"`
}

func (config TOMLV1) GetFileExtensionsOverlay() Overlay {
	return Overlay(config)
}

func (config *TOMLV1) Reset() {
	equals.SetIfNotNil(config.Conflict, "")
	equals.SetIfNotNil(config.Lockfile, "")
	equals.SetIfNotNil(config.Organize, "")
	equals.SetIfNotNil(config.Repo, "")
	equals.SetIfNotNil(config.Tag, "")
	equals.SetIfNotNil(config.Type, "")
	equals.SetIfNotNil(config.Zettel, "")
}

func (config *TOMLV1) ResetWith(b TOMLV1) {
	config.Config = b.Config
	config.Conflict = b.Conflict
	config.Lockfile = b.Lockfile
	config.Organize = b.Organize
	config.Repo = b.Repo
	config.Tag = b.Tag
	config.Type = b.Type
	config.Zettel = b.Zettel
}

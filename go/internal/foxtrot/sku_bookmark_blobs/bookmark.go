package sku_bookmark_blobs

//go:generate tommy generate
type TomlBookmark struct {
	Url string `toml:"url"`
}

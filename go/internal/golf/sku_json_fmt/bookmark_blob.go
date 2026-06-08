package sku_json_fmt

import (
	sku_bookmark_blobs "code.linenisgreat.com/dodder/go/internal/foxtrot/sku_bookmark_blobs"
)

type (
	TomlBookmark         = sku_bookmark_blobs.TomlBookmark
	TomlBookmarkDocument = sku_bookmark_blobs.TomlBookmarkDocument
)

var DecodeTomlBookmark = sku_bookmark_blobs.DecodeTomlBookmark

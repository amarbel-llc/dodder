package repo_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

type (
	DefaultsGetter interface {
		GetDefaults() Defaults
	}

	Defaults interface {
		GetDefaultType() ids.TypeStruct
		GetDefaultTags() collections_slice.Slice[ids.TagStruct]
	}
)

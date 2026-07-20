package sku

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
)

type Edges struct {
	Objects []ids.ObjectId
	Blobs   []markl.Id
	Skipped []error
}

type EdgeExplorer interface {
	ExploreEdges(object *Transacted) (Edges, error)
}

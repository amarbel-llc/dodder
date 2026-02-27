package tag_paths

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/bravo/collections_slice"
)

type (
	PathsWithTypes collections_slice.Slice[*PathWithType]
)

func (pathsWithTypes *PathsWithTypes) GetSlice() collections_slice.Slice[*PathWithType] {
	return collections_slice.Slice[*PathWithType](*pathsWithTypes)
}

func (pathsWithTypes *PathsWithTypes) GetSliceMutable() *collections_slice.Slice[*PathWithType] {
	return (*collections_slice.Slice[*PathWithType])(pathsWithTypes)
}

func (pathsWithTypes *PathsWithTypes) Reset() {
	pathsWithTypes.GetSliceMutable().Reset()
}

func (pathsWithTypes PathsWithTypes) Len() int {
	return len(pathsWithTypes)
}

func (pathsWithTypes PathsWithTypes) Less(left, right int) bool {
	return pathsWithTypes[left].Compare(&pathsWithTypes[right].Path).IsLess()
}

func (pathsWithTypes PathsWithTypes) ContainsPath(p *PathWithType) (int, bool) {
	return cmp.BinarySearchFuncIndex(
		pathsWithTypes,
		p,
		func(ep *PathWithType, el *PathWithType) cmp.Result {
			return ep.Compare(&p.Path)
		},
	)
}

func (pathsWithTypes *PathsWithTypes) AddNonEmptyPath(p *PathWithType) {
	if p == nil {
		return
	}

	pathsWithTypes.AddPath(p)
}

func (pathsWithTypes *PathsWithTypes) AddPath(p *PathWithType) (idx int, alreadyExists bool) {
	if p.IsEmpty() {
		return idx, alreadyExists
	}

	// p = p.Clone()

	idx, alreadyExists = pathsWithTypes.ContainsPath(p)

	if alreadyExists {
		return idx, alreadyExists
	}

	pathsWithTypes.GetSliceMutable().Insert(idx, p)

	return idx, alreadyExists
}

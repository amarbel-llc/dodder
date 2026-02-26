package directory_layout

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
)

type v3 struct {
	xdg XDG
}

var (
	_ Repo              = v3{}
	_ repoUninitialized = &v3{}
)

func (layout *v3) initialize(
	xdg XDG,
) (err error) {
	layout.xdg = xdg
	return err
}

func (layout v3) makeDirOrPanic(
	dirLayout interfaces.DirectoryLayoutBaseEnvVar,
	targets ...string,
) interfaces.DirectoryLayoutPath {
	if dirLayout.GetBaseEnvVarValue() == "" {
		panic(
			fmt.Sprintf(
				"empty xdg %q dir: %#v",
				dirLayout.GetBaseEnvVarName(),
				dirLayout,
			),
		)
	}

	return dirLayout.MakePath(targets...)
}

func (layout v3) MakeDirData(targets ...string) interfaces.DirectoryLayoutPath {
	return layout.makeDirOrPanic(layout.xdg.GetDirData(), targets...)
}

func (layout v3) MakeDirConfig(
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.makeDirOrPanic(layout.xdg.GetDirConfig(), targets...)
}

func (layout v3) MakeDirCache(
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.makeDirOrPanic(layout.xdg.GetDirCache(), targets...)
}

func (layout v3) MakePathBlobStore(
	targets ...string,
) interfaces.DirectoryLayoutPath {
	return layout.xdg.GetDirData().MakePath(
		stringSliceJoin("blob_stores", targets)...)
}

func (layout v3) FileCacheDormant() string {
	return layout.MakeDirData("dormant").String()
}

func (layout v3) FileTags() string {
	return layout.MakeDirData("tags").String()
}

func (layout v3) FileLock() string {
	return layout.xdg.GetDirState().MakePath("lock").String()
}

func (layout v3) FileConfig() string {
	return layout.MakeDirConfig("config-mutable").String()
}

func (layout v3) DirDataIndex(p ...string) string {
	return layout.MakeDirData(stringSliceJoin("index", p)...).String()
}

func (layout v3) DirCacheRepo(p ...string) string {
	return layout.xdg.GetDirCache().MakePath(
		stringSliceJoin("repo", p)...).String()
}

func (layout v3) DirLostAndFound() string {
	return layout.MakeDirCache("lost_and_found").String()
}

func (layout v3) DirIndexObjects() string {
	return layout.DirDataIndex("objects")
}

func (layout v3) DirIndexObjectPointers() string {
	return layout.DirDataIndex("object_pointers/Page")
}

func (layout v3) DirCacheRemoteInventoryListsLog() string {
	return layout.xdg.GetDirCache().MakePath(
		"remote_inventory_lists_log",
	).String()
}

func (layout v3) DirObjectId() string {
	return layout.MakeDirData("object_ids").String()
}

func (layout v3) FileObjectIdLog() string {
	return layout.MakeDirData("object_id_log").String()
}

func (layout v3) FileCacheObjectId() string {
	return layout.DirDataIndex("object_id")
}

func (layout v3) FileInventoryListLog() string {
	return layout.MakeDirData("inventory_lists_log").String()
}

func (layout v3) DirsGenesis() []string {
	return []string{
		layout.MakeDirConfig().String(),
		layout.DirObjectId(),
		layout.DirDataIndex(),
		layout.DirLostAndFound(),
	}
}

// Deprecated

// TODO deprecate and remove
func (layout v3) DirFirstBlobStoreInventoryLists() string {
	panic(errors.Err405MethodNotAllowed)
}

// TODO deprecate and remove
func (layout v3) DirFirstBlobStoreBlobs() string {
	panic(errors.Err405MethodNotAllowed)
}

func (layout v3) GetLocationType() blob_store_id.LocationType {
	return layout.xdg.GetLocationType()
}

func (layout v3) cloneUninitialized() uninitializedXDG {
	return &layout
}

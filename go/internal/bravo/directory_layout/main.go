package directory_layout

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	madder_dl "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
)

type (
	XDG = xdg.XDG

	BlobStore = madder_dl.BlobStore
	Common    = madder_dl.Common

	Repo interface {
		MakeDirData(p ...string) interfaces.DirectoryLayoutPath

		DirDataIndex(p ...string) string
		DirCacheRemoteInventoryListsLog() string
		DirIndexObjectPointers() string
		DirIndexObjects() string

		DirCacheRepo(p ...string) string

		DirLostAndFound() string
		DirObjectId() string

		FileCacheDormant() string
		FileCacheObjectId() string
		FileConfig() string
		FileConfigTags() string
		FileConfigTypes() string
		FileConfigRepos() string
		FileLock() string
		FileTags() string
		FileInventoryListLog() string
		FileZettelIdLog() string

		DirsGenesis() []string
	}

	Mutable interface {
		Delete(...string) error
	}

	RepoMutable interface {
		Repo
		Mutable
	}
)

type repoUninitialized interface {
	Repo
	initialize(XDG) error
}

func MakeRepo(
	storeVersion store_version.Version,
	xdg XDG,
) (Repo, error) {
	var repo repoUninitialized = &v3{}

	if err := repo.initialize(xdg); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return repo, nil
}

func MakeBlobStore(
	storeVersion store_version.Version,
	xdg XDG,
) (BlobStore, error) {
	return madder_dl.MakeBlobStore(xdg)
}

func CloneBlobStoreWithXDG(layout BlobStore, xdg XDG) (BlobStore, error) {
	return madder_dl.CloneBlobStoreWithXDG(layout, xdg)
}

package directory_layout

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
)

type (
	XDG = xdg.XDG

	Common interface {
		blob_store_id.LocationTypeGetter
		cloneUninitialized() uninitializedXDG
	}

	BlobStore interface {
		Common
		MakePathBlobStore(...string) interfaces.DirectoryLayoutPath
	}

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

type (
	uninitializedXDG interface {
		BlobStore
		Repo
		initialize(XDG) error
	}

	blobStoreUninitialized interface {
		BlobStore
		uninitializedXDG
	}

	repoUninitialized interface {
		Repo
		uninitializedXDG
	}
)

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
	var blobStore blobStoreUninitialized = &v3{}

	if err := blobStore.initialize(xdg); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return blobStore, nil
}

func CloneBlobStoreWithXDG(layout BlobStore, xdg XDG) (BlobStore, error) {
	clone := layout.cloneUninitialized()

	if err := clone.initialize(xdg); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return clone, nil
}

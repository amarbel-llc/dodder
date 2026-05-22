package directory_layout

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/xdg"
)

type (
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
	initialize(xdg.XDG) error
}

func MakeRepo(
	storeVersion store_version.Version,
	x xdg.XDG,
) (Repo, error) {
	var repo repoUninitialized = &v3{}

	if err := repo.initialize(x); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return repo, nil
}

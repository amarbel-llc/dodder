package zettel_id_index

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/hotel/genesis_configs"
	hinweis_index_v0 "code.linenisgreat.com/dodder/go/internal/india/zettel_id_index/v0"
	hinweis_index_v1 "code.linenisgreat.com/dodder/go/internal/india/zettel_id_index/v1"
)

type Index interface {
	errors.Flusher
	CreateZettelId() (*ids.ZettelId, error)
	interfaces.ResetableWithError
	AddZettelId(ids.Id) error
	PeekZettelIds(int) ([]*ids.ZettelId, error)
}

func MakeIndex(
	config genesis_configs.ConfigPublic,
	configCli repo_config_cli.Config,
	directoryLayout directory_layout.RepoMutable,
	cacheIOFactory domain_interfaces.NamedBlobAccess,
) (i Index, err error) {
	if false {
		ui.TodoP3("investigate using bitsets")
		if i, err = hinweis_index_v1.MakeIndex(
			configCli,
			directoryLayout,
			cacheIOFactory,
		); err != nil {
			err = errors.Wrap(err)
			return i, err
		}

	} else {
		if i, err = hinweis_index_v0.MakeIndex(
			configCli,
			directoryLayout,
			cacheIOFactory,
		); err != nil {
			err = errors.Wrap(err)
			return i, err
		}
	}

	return i, err
}

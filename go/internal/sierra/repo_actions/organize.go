package repo_actions

import (
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	papa_repo "code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/lib/0/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// TODO migrate over to Organize2
type Organize struct {
	*repo
	orgie.Metadata
	DontUseQueryGroupForOrganizeMetadata bool
}

func (op Organize) RunWithQueryGroup(
	qg *queries.Query,
) (organizeResults orgie.OrganizeResults, err error) {
	skus := sku.MakeSkuTypeSetMutable()
	var l sync.RWMutex

	if err = op.GetStore().QueryTransactedAsSkuType(
		qg,
		func(co sku.SkuType) (err error) {
			l.Lock()
			defer l.Unlock()

			cloned, _ := co.Clone() //repool:owned
			return skus.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	if organizeResults, err = op.RunWithSkuType(qg, skus); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	return organizeResults, err
}

// RunWithRemoteQueryGroup resolves qg against remote (not the local store),
// then runs the same organize outline/diff flow as RunWithTransacted. Used
// for pre-pull filtering (clone, init-workspace): the objects don't exist
// locally yet, so there is nothing local to query against.
func (op Organize) RunWithRemoteQueryGroup(
	remote papa_repo.Repo,
	qg *queries.Query,
) (organizeResults orgie.OrganizeResults, err error) {
	var list *sku.HeapTransacted

	if list, err = remote.MakeInventoryList(qg); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	transacted := sku.MakeTransactedMutableSet()

	for object := range list.All() {
		transacted.Add(object)
	}

	if organizeResults, err = op.RunWithTransacted(qg, transacted); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	return organizeResults, err
}

// TODO remove
func (op Organize) RunWithTransacted(
	qg *queries.Query,
	transacted sku.TransactedSet,
) (organizeResults orgie.OrganizeResults, err error) {
	skus := sku.MakeSkuTypeSetMutable()

	for z := range transacted.All() {
		clone, _ := sku.CloneSkuTypeFromTransacted( //repool:owned
			z.GetSku(),
			checked_out_state.Internal,
		)

		skus.Add(clone)
	}

	if organizeResults, err = op.RunWithSkuType(qg, skus); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	return organizeResults, err
}

func (op Organize) RunWithSkuType(
	q *queries.Query,
	skus sku.SkuTypeSet,
) (organizeResults orgie.OrganizeResults, err error) {
	organizeResults.Original = skus
	organizeResults.QueryGroup = q

	var repoId ids.RepoId

	if q != nil {
		repoId = q.RepoId
	}

	if organizeResults.QueryGroup == nil ||
		op.DontUseQueryGroupForOrganizeMetadata {
		b := op.MakeQueryBuilder(
			ids.MakeGenre(genres.All()...),
			nil,
		).WithExternalLike(
			skus,
		)

		if organizeResults.QueryGroup, err = b.BuildQueryGroup(); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}
	}

	organizeResults.QueryGroup.RepoId = repoId

	organizeFlags := orgie.MakeFlagsWithMetadata(op.Metadata)
	ApplyToOrganizeOptions(op.repo, &organizeFlags.Options)
	organizeFlags.Skus = skus

	createOrganizeFileOp := MakeCreateOrganizeFile(
		op.repo,
		MakeOrganizeOptionsWithQueryGroup(
			op.repo,
			organizeFlags,
			organizeResults.QueryGroup,
		),
	)

	types := queries.GetTypes(organizeResults.QueryGroup)

	if types.Len() == 1 {
		createOrganizeFileOp.Type = quiter_set.Any(types)
	}

	var file *os.File

	if file, err = op.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
		"*." + op.GetConfig().GetFileExtensions().Organize,
	); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	defer errors.DeferredCloser(&err, file)

	if organizeResults.Before, err = createOrganizeFileOp.RunAndWrite(
		file,
	); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	// TODO refactor into common vim processing loop
	for {
		openVimOp := MakeOpenEditor(op.repo)
		openVimOp.VimOptions = vim_cli_options_builder.New().
			WithFileType("dodder-organize").
			Build()

		if err = openVimOp.Run(file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		// if err = op.Reset(); err != nil {
		// 	err = errors.Wrap(err)
		// 	return
		// }

		// Reopen by path rather than seeking on the original handle: an
		// editor that saves via the common rename-over-original-path
		// idiom (e.g. vim's default backupcopy=auto) replaces the inode
		// at file.Name(), leaving the original *os.File pointing at the
		// old, unlinked-but-still-open inode. Seeking and reading from
		// that stale handle silently returns the pre-edit content.
		var reopened *os.File

		if reopened, err = os.Open(file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		readOrganizeTextOp := MakeReadOrganizeFile(op.repo)

		organizeResults.After, err = readOrganizeTextOp.Run(
			reopened,
			orgie.NewMetadataWithSettingLookup(
				organizeResults.Before.GetRepoId(),
				op.GetPrototypeSettings(),
			),
		)

		errors.PanicIfError(reopened.Close())

		if err != nil {
			if op.handleReadChangesError(op.repo, err) {
				err = nil
				continue
			} else {
				ui.Err().Printf("aborting organize")
				return organizeResults, err
			}
		}

		break
	}

	return organizeResults, err
}

func (cmd Organize) handleReadChangesError(
	envUI env_ui.Env,
	err error,
) (tryAgain bool) {
	var errorRead orgie.ErrorRead

	if err != nil && !errors.As(err, &errorRead) {
		ui.Err().Printf("unrecoverable organize read failure: %s", err)
		tryAgain = false
		return tryAgain
	}

	return envUI.Retry(
		"reading changes failed",
		"edit and try again?",
		err,
	)
}

package user_ops

import (
	"io"
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

// TODO migrate over to Organize2
type Organize struct {
	*local_working_copy.Repo
	organize_text.Metadata
	DontUseQueryGroupForOrganizeMetadata bool
}

func (op Organize) RunWithQueryGroup(
	qg *queries.Query,
) (organizeResults organize_text.OrganizeResults, err error) {
	skus := sku.MakeSkuTypeSetMutable()
	var l sync.RWMutex

	if err = op.GetStore().QueryTransactedAsSkuType(
		qg,
		func(co sku.SkuType) (err error) {
			l.Lock()
			defer l.Unlock()

			cloned, _ := co.Clone()
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

// TODO remove
func (op Organize) RunWithTransacted(
	qg *queries.Query,
	transacted sku.TransactedSet,
) (organizeResults organize_text.OrganizeResults, err error) {
	skus := sku.MakeSkuTypeSetMutable()

	for z := range transacted.All() {
		clone, _ := sku.CloneSkuTypeFromTransacted(
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
) (organizeResults organize_text.OrganizeResults, err error) {
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

	organizeFlags := organize_text.MakeFlagsWithMetadata(op.Metadata)
	op.ApplyToOrganizeOptions(&organizeFlags.Options)
	organizeFlags.Skus = skus

	createOrganizeFileOp := CreateOrganizeFile{
		Repo: op.Repo,
		Options: op.Repo.MakeOrganizeOptionsWithQueryGroup(
			organizeFlags,
			organizeResults.QueryGroup,
		),
	}

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
		openVimOp := OpenEditor{
			VimOptions: vim_cli_options_builder.New().
				WithFileType("dodder-organize").
				Build(),
		}

		if err = openVimOp.Run(op.Repo, file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		// if err = op.Reset(); err != nil {
		// 	err = errors.Wrap(err)
		// 	return
		// }

		readOrganizeTextOp := ReadOrganizeFile{}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		if organizeResults.After, err = readOrganizeTextOp.Run(
			op.Repo,
			file,
			organize_text.NewMetadataWithOptionCommentLookup(
				organizeResults.Before.GetRepoId(),
				op.GetPrototypeOptionComments(),
			),
		); err != nil {
			if op.handleReadChangesError(op.Repo, err) {
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
	var errorRead organize_text.ErrorRead

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

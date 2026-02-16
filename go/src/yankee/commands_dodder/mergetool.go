package commands_dodder

import (
	"bufio"
	"os"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("merge-tool", &Mergetool{})
}

type Mergetool struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
}

func (cmd Mergetool) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	envWorkspace := localWorkingCopy.GetEnvWorkspace()
	envWorkspace.AssertNotTemporary(req)

	conflicted := sku.MakeSkuTypeSetMutable()

	if err := localWorkingCopy.GetStore().QuerySkuType(
		queryGroup,
		func(co sku.SkuType) (err error) {
			if co.GetState() != checked_out_state.Conflicted {
				return err
			}

			cloned, _ := co.Clone()
			if err = conflicted.Add(cloned); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock),
	)

	if conflicted.Len() == 0 {
		// TODO-P2 return status 1 and use Err
		localWorkingCopy.GetUI().Printf("nothing to merge")
		return
	}

	for co := range conflicted.All() {
		cmd.doOne(localWorkingCopy, co)
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)
}

func (cmd Mergetool) doOne(
	repo *local_working_copy.Repo,
	checkedOut *sku.CheckedOut,
) {
	conflicted := sku.Conflicted{
		CheckedOut: func() *sku.CheckedOut { c, _ := checkedOut.Clone(); return c }(),
	}

	var conflict *fd.FD

	{
		var err error

		if conflict, err = repo.GetEnvWorkspace().GetStoreFS().GetConflictOrError(
			checkedOut.GetSkuExternal(),
		); err != nil {
			repo.Cancel(err)
		}
	}

	var file *os.File

	{
		var err error

		if file, err = files.Open(conflict.GetPath()); err != nil {
			repo.Cancel(err)
			return
		}

		defer errors.ContextMustClose(repo, file)
	}

	// TODO pool
	bufferedReader := bufio.NewReader(file)

	inventoryListCoderCloset := repo.GetInventoryListCoderCloset()

	if err := conflicted.ReadConflictMarker(
		inventoryListCoderCloset.IterInventoryListBlobSkusFromReader(
			ids.DefaultOrPanic(genres.InventoryList),
			bufferedReader,
		),
	); err != nil {
		repo.Cancel(err)
	}

	if err := repo.GetStore().RunMergeTool(
		conflicted,
	); err != nil {
		repo.Cancel(err)
	}
}

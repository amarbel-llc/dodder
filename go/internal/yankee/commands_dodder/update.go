package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd(
		"update",
		&Update{},
	)
}

type Update struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.Query
}

var _ interfaces.CommandComponentWriter = (*Update)(nil)

func (cmd *Update) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
}

func (cmd Update) Run(req command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	args := req.PopArgs()

	query := cmd.MakeQueryIncludingWorkspace(
		req,
		pkg_query.BuilderOptions(
			pkg_query.BuilderOptionWorkspace(localWorkingCopy),
			pkg_query.BuilderOptionDefaultGenres(genres.Zettel),
		),
		localWorkingCopy,
		args,
	)

	store := localWorkingCopy.GetStore()

	req.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock))

	// TODO fix issue with non-deterministic query causing ordering issues
	if err := store.QueryTransacted(
		query,
		func(object *sku.Transacted) (err error) {
			var typeObject *sku.Transacted

			if typeObject, err = store.ReadOneObjectId(object.GetType()); err != nil {
				err = errors.Wrap(err)
				return err
			}

			object.GetMetadataMutable().GetTypeLockMutable().GetValueMutable().ResetWithMarklId(
				typeObject.GetMetadata().GetObjectSig(),
			)

			if err = store.CreateOrUpdate(
				object,
				sku.CommitOptions{},
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}

	req.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock))
}

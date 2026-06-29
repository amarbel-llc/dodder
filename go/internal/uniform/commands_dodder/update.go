package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"update",
		&Update{},
	)
}

func (cmd Update) GetDescription() command.Description {
	return command.Description{
		Short: "update type lock signatures",
	}
}

type Update struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.Query
}

var (
	_ interfaces.CommandComponentWriter = (*Update)(nil)
	_ command.CommandWithArgs           = (*Update)(nil)
)

func (cmd *Update) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

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

			// MergeCheckedOut refreshes a clean checked-out copy to the
			// re-stamped state so `dodder update` does not leave checkouts
			// stale; a no-op when the object is not checked out. Mirrors
			// UpdateObject / organize / revert.
			if err = store.CreateOrUpdate(
				object,
				sku.CommitOptions{
					StoreOptions: sku.StoreOptions{MergeCheckedOut: true},
				},
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

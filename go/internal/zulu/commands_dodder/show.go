package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	pkg_query "code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/repo"
	"code.linenisgreat.com/dodder/go/internal/whiskey/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func init() {
	utility.AddCmd("show", &Show{
		Format: local_working_copy.FormatFlag{
			DefaultFormatter: local_working_copy.GetFormatFuncConstructorEntry(
				"log",
			),
		},
	})
}

type Show struct {
	command_components_dodder.EnvRepo
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.Query
	command_components_dodder.RemoteTransfer

	complete command_components_dodder.Complete

	After      ids.Tai
	Before     ids.Tai
	Format     local_working_copy.FormatFlag
	RemoteRepo ids.RepoId
}

var _ interfaces.CommandComponentWriter = (*Show)(nil)

func (cmd *Show) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)
	cmd.Query.SetFlagDefinitions(flagSet)

	flagSet.Var(
		&cmd.Format,
		"format",
		"format used when outputting objects to stdout",
	)

	flagSet.Var((*ids.TaiRFC3339Value)(&cmd.Before), "before", "")
	flagSet.Var((*ids.TaiRFC3339Value)(&cmd.After), "after", "")
	flagSet.Var(&cmd.RemoteRepo, "repo", "the remote repo to query")
}

func (cmd Show) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	repo := cmd.MakeLocalWorkingCopy(req)

	args := commandLine.FlagsOrArgs[1:]

	if commandLine.InProgress != "" {
		args = args[:len(args)-1]
	}

	cmd.complete.CompleteObjects(
		req,
		repo,
		pkg_query.BuilderOptionDefaultGenres(genres.Tag),
		args...,
	)
}

func (cmd Show) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	args := req.PopArgs()

	query := cmd.MakeQueryIncludingWorkspace(
		req,
		pkg_query.BuilderOptions(
			pkg_query.BuilderOptionWorkspace(repo),
			pkg_query.BuilderOptionDefaultGenres(genres.Zettel),
		),
		repo,
		args,
	)

	cmd.runWithLocalWorkingCopyAndQuery(req, repo, query)
}

func (cmd Show) runWithLocalWorkingCopyAndQuery(
	req command.Request,
	localWorkingCopy *local_working_copy.Repo,
	query *pkg_query.Query,
) {
	var remoteObject *sku.Transacted
	var remoteWorkingCopy repo.Repo

	if !cmd.RemoteRepo.IsEmpty() {
		var err error

		if remoteObject, err = localWorkingCopy.GetObjectFromObjectId(
			cmd.RemoteRepo.StringWithSlashPrefix(),
		); err != nil {
			localWorkingCopy.Cancel(err)
		}

		remoteRepo := cmd.MakeRemote(req, localWorkingCopy, remoteObject)
		remoteWorkingCopy, _ = remoteRepo.(repo.Repo)
	}

	if cmd.Format.GetName() == "" && pkg_query.IsExactlyOneObjectId(query) {
		// if err := cmd.Format.Set("text"); err != nil {
		// 	localWorkingCopy.Cancel(err)
		// 	return
		// }
	}

	output := cmd.Format.MakeFormatFunc(
		localWorkingCopy,
		localWorkingCopy.GetUIFile(),
	)

	if !cmd.Before.IsEmpty() {
		old := output

		output = func(object *sku.Transacted) (err error) {
			if !object.GetTai().Before(cmd.Before) {
				return err
			}

			if err = old(object); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		}
	}

	if !cmd.After.IsEmpty() {
		old := output

		output = func(object *sku.Transacted) (err error) {
			if !object.GetTai().After(cmd.After) {
				return err
			}

			if err = old(object); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		}
	}

	if remoteWorkingCopy != nil {
		var list *sku.HeapTransacted

		{
			var err error

			if list, err = remoteWorkingCopy.MakeInventoryList(query); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}

		for object := range list.All() {
			if err := output(object); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}
	} else {
		if err := localWorkingCopy.GetStore().QueryTransacted(
			query,
			quiter.MakeSyncSerializer(output),
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}
}

package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("last", &Last{
		Format: local_working_copy.FormatFlag{
			DefaultFormatter: local_working_copy.GetFormatFuncConstructorEntry(
				"box",
			),
		},
	})
}

func (cmd Last) GetDescription() command.Description {
	return command.Description{
		Short: "display the most recently committed objects",
	}
}

type Last struct {
	command_components_dodder.InventoryLists
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.EnvRepo

	RepoId   ids.RepoId
	Edit     bool
	Organize bool
	Format   local_working_copy.FormatFlag
}

var _ interfaces.CommandComponentWriter = (*Last)(nil)

func (cmd *Last) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagSet)

	// TODO remove
	flagSet.Var(&cmd.RepoId, "kasten", "none or Browser")
	flagSet.Var(&cmd.Format, "format", "format")
	flagSet.BoolVar(&cmd.Organize, "organize", false, "")
	flagSet.BoolVar(&cmd.Edit, "edit", false, "")
}

func (cmd Last) CompletionGenres() ids.Genre {
	return ids.MakeGenre(
		genres.InventoryList,
	)
}

func (cmd Last) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	if len(req.PopArgs()) != 0 {
		ui.Err().Print("ignoring arguments")
	}

	cmd.runLocalWorkingCopy(repo)
}

func (cmd Last) runLocalWorkingCopy(localWorkingCopy *local_working_copy.Repo) {
	if (cmd.Edit || cmd.Organize) && cmd.Format.WasSet() {
		ui.Err().Print("ignoring format")
	} else if cmd.Edit && cmd.Organize {
		errors.ContextCancelWithErrorf(localWorkingCopy, "cannot organize and edit at the same time")
	}

	skus := sku.MakeTransactedMutableSet()

	var funcIter interfaces.FuncIter[*sku.Transacted]

	if cmd.Organize || cmd.Edit {
		funcIter = skus.Add
	} else {
		funcIter = cmd.Format.MakeFormatFunc(
			localWorkingCopy,
			localWorkingCopy.GetUIFile(),
		)
	}

	funcIter = quiter.MakeSyncSerializer(funcIter)

	if err := cmd.runWithInventoryList(
		localWorkingCopy.GetEnvRepo(),
		localWorkingCopy,
		funcIter,
	); err != nil {
		localWorkingCopy.GetEnvRepo().Cancel(err)
		return
	}

	if cmd.Organize {
		opOrganize := repo_actions.MakeOrganize(
			localWorkingCopy,
			orgie.Metadata{
				OptionCommentSet: orgie.MakeOptionCommentSet(nil),
			},
		)

		var results orgie.OrganizeResults

		{
			var err error

			if results, err = opOrganize.RunWithTransacted(nil, skus); err != nil {
				localWorkingCopy.GetEnvRepo().Cancel(err)
			}
		}

		if _, err := repo_actions.LockAndCommitOrganizeResults(localWorkingCopy, results); err != nil {
			localWorkingCopy.GetEnvRepo().Cancel(err)
		}
	} else if cmd.Edit {
		opCheckout := repo_actions.MakeCheckout(localWorkingCopy)
		opCheckout.Options = checkout_options.Options{
			CheckoutMode: checkout_mode.Make(checkout_mode.MetadataAndBlob),
		}
		opCheckout.Edit = true

		if _, err := opCheckout.Run(skus); err != nil {
			localWorkingCopy.GetEnvRepo().Cancel(err)
		}
	}
}

func (cmd Last) runWithInventoryList(
	envRepo env_repo.Env,
	repo repo.Repo,
	funcIter interfaces.FuncIter[*sku.Transacted],
) (err error) {
	var listObject *sku.Transacted

	inventoryListStore := repo.GetInventoryListStore()

	if listObject, err = inventoryListStore.ReadLast(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	inventoryListBlobStore := cmd.MakeInventoryListCoderCloset(
		envRepo,
	)

	seq := inventoryListBlobStore.StreamInventoryListBlobSkus(
		listObject,
	)

	for object, seqError := range seq {
		if seqError != nil {
			ui.CLIErrorTreeEncoder.EncodeTo(seqError, ui.Err())
			continue
		}

		func() {
			// TODO investigate the pool folow for StreamInventoryListBlobSkus
			// and
			// determine why repooling here is breaking things
			// defer sku.GetTransactedPool().Put(sk)

			if err = funcIter(object); err != nil {
				err = errors.Wrap(err)
				return
			}
		}()
	}

	return err
}

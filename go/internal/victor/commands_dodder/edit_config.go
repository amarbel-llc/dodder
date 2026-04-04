package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd("edit-config", &EditConfig{})
}

type EditConfig struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*EditConfig)(nil)

func (cmd *EditConfig) GetArgs() []command.ArgGroup { return nil }

func (cmd EditConfig) GetDescription() command.Description {
	return command.Description{
		Short: "edit the repository configuration",
	}
}

func (cmd EditConfig) Run(
	req command.Request,
) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command edit-konfig ignores passed in arguments.")
	}

	var sk *sku.Transacted

	{
		var err error

		if sk, err = cmd.editInVim(localWorkingCopy); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Reset),
	)

	builder := import_plan.MakeLocalBuilder()
	if err := builder.AddObject(sk, 0); err != nil {
		localWorkingCopy.Cancel(err)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		localWorkingCopy.Cancel(buildErr)
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: localWorkingCopy.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
		},
	}

	if _, err := localWorkingCopy.ExecutePlan(plan); err != nil {
		localWorkingCopy.Cancel(err)
	}
}

func (cmd EditConfig) editInVim(
	repo *local_working_copy.Repo,
) (sk *sku.Transacted, err error) {
	var file *os.File

	if file, err = repo.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
		fmt.Sprintf("*.%s", repo.GetConfig().GetFileExtensions().Config),
	); err != nil {
		err = errors.Wrap(err)
		return sk, err
	}

	path := file.Name()

	if err = file.Close(); err != nil {
		err = errors.Wrap(err)
		return sk, err
	}

	if err = cmd.makeTempConfigFile(repo, path); err != nil {
		err = errors.Wrap(err)
		return sk, err
	}

	openVimOp := repo_actions.MakeOpenEditor(repo)
	openVimOp.VimOptions = vim_cli_options_builder.New().
		WithFileType("dodder-object").
		Build()

	if err = openVimOp.Run(path); err != nil {
		err = errors.Wrap(err)
		return sk, err
	}

	if sk, err = cmd.readTempConfigFile(repo, path); err != nil {
		err = errors.Wrap(err)
		return sk, err
	}

	return sk, err
}

func (cmd EditConfig) makeTempConfigFile(
	repo *local_working_copy.Repo,
	path string,
) (err error) {
	var configObject *sku.Transacted

	if configObject, err = repo.GetStore().ReadTransactedFromObjectId(
		ids.Config,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var fsItem sku.FSItem
	fsItem.Reset()

	if err = fsItem.Object.Set(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	fsItem.FDs.Add(&fsItem.Object)

	if err = repo.GetEnvWorkspace().GetStoreFS().GetFileEncoder().Encode(
		checkout_options.TextFormatterOptions{},
		configObject,
		&fsItem,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (cmd EditConfig) readTempConfigFile(
	localWorkingCopy *local_working_copy.Repo,
	path string,
) (object *sku.Transacted, err error) {
	object, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

	if object.GetObjectIdMutable().Set("konfig"); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	var file *os.File

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	defer errors.DeferredCloser(&err, file)

	if err = localWorkingCopy.GetEnvWorkspace().GetStoreFS().ReadOneExternalObjectReader(
		file,
		object,
	); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

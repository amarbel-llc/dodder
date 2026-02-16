package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/src/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("edit-config", &EditConfig{})
}

type EditConfig struct {
	command_components_dodder.LocalWorkingCopy
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

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock),
	)

	if err := localWorkingCopy.GetStore().CreateOrUpdateDefaultProto(
		sk,
		sku.StoreOptions{},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)
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

	openVimOp := user_ops.OpenEditor{
		VimOptions: vim_cli_options_builder.New().
			WithFileType("dodder-object").
			Build(),
	}

	if err = openVimOp.Run(repo, path); err != nil {
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
	object, _ = sku.GetTransactedPool().GetWithRepool()

	if object.ObjectId.Set("konfig"); err != nil {
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

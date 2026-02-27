package commands_dodder

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("dormant-edit", &DormantEdit{})
}

type DormantEdit struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd DormantEdit) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command dormant-edit ignores passed in arguments.")
	}

	var digest domain_interfaces.MarklId

	{
		var err error

		if digest, err = cmd.editInVim(localWorkingCopy); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}

	if err := localWorkingCopy.Reset(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	if err := localWorkingCopy.Lock(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	defer localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)

	if _, err := localWorkingCopy.GetStore().UpdateKonfig(digest); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}
}

// TODO refactor into common
func (cmd DormantEdit) editInVim(
	repo *local_working_copy.Repo,
) (digest domain_interfaces.MarklId, err error) {
	var path string

	if path, err = cmd.makeTempFile(repo); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	openVimOp := user_ops.OpenEditor{
		VimOptions: vim_cli_options_builder.New().
			Build(),
	}

	if err = openVimOp.Run(repo, path); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	if digest, err = cmd.readTempFile(repo, path); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	return digest, err
}

// TODO refactor into common
func (cmd DormantEdit) makeTempFile(
	repo *local_working_copy.Repo,
) (path string, err error) {
	var object *sku.Transacted

	if object, err = repo.GetStore().ReadTransactedFromObjectId(
		ids.Config,
	); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	var file *os.File

	if file, err = repo.GetEnvRepo().GetTempLocal().FileTemp(); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	defer errors.DeferredCloser(&err, file)

	var readCloser io.ReadCloser

	if readCloser, err = repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	path = file.Name()

	if _, err = ohio.CopyBuffered(file, readCloser); err != nil {
		err = errors.Wrap(err)
		return path, err
	}

	return path, err
}

// TODO refactor into common
func (cmd DormantEdit) readTempFile(
	repo *local_working_copy.Repo,
	path string,
) (digest domain_interfaces.MarklId, err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	defer errors.DeferredCloser(&err, file)

	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	var typedBlob repo_configs.TypedBlob

	coder := repo.GetStore().GetConfigBlobCoder()

	// TODO-P3 offer option to edit again
	if _, err = coder.DecodeFrom(
		&typedBlob,
		io.TeeReader(file, writeCloser),
	); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	// TODO persist blob type

	digest = writeCloser.GetMarklId()

	return digest, err
}

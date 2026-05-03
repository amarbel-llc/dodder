package commands_dodder

import (
	"io"
	"os"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/0/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

func init() {
	utility.AddCmd("dormant-edit", &DormantEdit{})
}

func (cmd DormantEdit) GetDescription() command.Description {
	return command.Description{
		Short: "edit dormant tags in an editor",
	}
}

type DormantEdit struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*DormantEdit)(nil)

// GetArgs returns nil: dormant-edit pops args but ignores them with a warning.
func (cmd *DormantEdit) GetArgs() []command.ArgGroup { return nil }

func (cmd DormantEdit) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args) > 0 {
		ui.Err().Print("Command dormant-edit ignores passed in arguments.")
	}

	var digest mad_domain_interfaces.MarklId

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
) (digest mad_domain_interfaces.MarklId, err error) {
	var path string

	if path, err = cmd.makeTempFile(repo); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	openVimOp := repo_actions.MakeOpenEditor(repo)
	openVimOp.VimOptions = vim_cli_options_builder.New().
		Build()

	if err = openVimOp.Run(path); err != nil {
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
) (digest mad_domain_interfaces.MarklId, err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	defer errors.DeferredCloser(&err, file)

	var writeCloser mad_domain_interfaces.BlobWriter

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

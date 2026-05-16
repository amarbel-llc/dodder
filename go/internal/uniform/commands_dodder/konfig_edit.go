package commands_dodder

import (
	"fmt"
	"io"
	"os"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/lib/0/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

// editKonfigInVim opens the konfig blob in $EDITOR and returns the
// new blob digest after the user's edit. The caller is responsible for
// the surrounding Lock / UpdateKonfig / Unlock dance. Shared by
// `edit-config` and `dormant-edit`.
//
// The blob is presented bare (no hyphence typed-blob wrapper); the
// konfig type is plumbed from the existing sku rather than parsed
// from the editor buffer.
func editKonfigInVim(
	repo *local_working_copy.Repo,
) (digest mad_domain_interfaces.MarklId, err error) {
	var object *sku.Transacted

	if object, err = repo.GetStore().ReadTransactedFromObjectId(
		ids.Config,
	); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	var path string

	if path, err = makeKonfigTempFile(repo, object); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	openVimOp := repo_actions.MakeOpenEditor(repo)
	openVimOp.VimOptions = vim_cli_options_builder.New().Build()

	if err = openVimOp.Run(path); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	if digest, err = readKonfigTempFile(repo, path, object.GetType()); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	return digest, err
}

// makeKonfigTempFile copies the current konfig blob bytes verbatim
// into a `*.<config-extension>` temp file. The extension is taken
// from the konfig itself so vim's ftdetect (zz-vim/ftdetect/
// dodder-object.vim) picks up the right syntax-highlighting filetype.
func makeKonfigTempFile(
	repo *local_working_copy.Repo,
	object *sku.Transacted,
) (path string, err error) {
	var file *os.File

	if file, err = repo.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
		fmt.Sprintf("*.%s", repo.GetConfig().GetFileExtensions().Config),
	); err != nil {
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

// readKonfigTempFile decodes the edited bytes back through the konfig
// type's body codec (rejecting outright TOML-parse failures), and at
// the same time pipes those bytes through to a fresh BlobWriter so
// the new content-addressed digest can be returned to the caller.
//
// configType is the konfig sku's existing type (e.g. !toml-config-v2);
// in the bare-blob model the editor buffer contains no type
// information, so the caller must plumb it in.
func readKonfigTempFile(
	repo *local_working_copy.Repo,
	path string,
	configType ids.Type,
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

	typedBlob := repo_configs.TypedBlob{
		Type: ids.MustTypeStruct(configType.String()).ToMadder(),
	}

	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(
		io.TeeReader(file, writeCloser),
	)
	defer repoolBufferedReader()

	if _, err = repo_configs.Coder.Blob.DecodeFrom(
		&typedBlob,
		bufferedReader,
	); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	digest = writeCloser.GetMarklId()

	return digest, err
}

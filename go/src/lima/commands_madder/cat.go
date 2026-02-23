package commands_madder

import (
	"io"
	"os/exec"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/delim_io"
	"code.linenisgreat.com/dodder/go/src/delta/script_value"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("cat", &Cat{})
}

type Cat struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreLocal

	Utility   script_value.Utility
	PrefixSha bool
}

var _ interfaces.CommandComponentWriter = (*Cat)(nil)

func (cmd Cat) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd *Cat) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(&cmd.Utility, "utility", "")
	flagSet.BoolVar(&cmd.PrefixSha, "prefix-sha", false, "")
}

type blobIdWithReadCloser struct {
	BlobId     domain_interfaces.MarklId
	ReadCloser io.ReadCloser
}

func (cmd Cat) makeBlobWriter(
	envRepo env_repo.BlobStoreEnv,
	blobStore blob_stores.BlobStoreInitialized,
) interfaces.FuncIter[blobIdWithReadCloser] {
	if cmd.Utility.IsEmpty() {
		return quiter.MakeSyncSerializer(
			func(readCloser blobIdWithReadCloser) (err error) {
				if err = cmd.copy(envRepo, blobStore, readCloser); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			},
		)
	} else {
		return quiter.MakeSyncSerializer(
			func(readCloser blobIdWithReadCloser) (err error) {
				defer errors.DeferredCloser(&err, readCloser.ReadCloser)

				utility := exec.Command(cmd.Utility.Head(), cmd.Utility.Tail()...)
				utility.Stdin = readCloser.ReadCloser

				var out io.ReadCloser

				if out, err = utility.StdoutPipe(); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if err = utility.Start(); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if err = cmd.copy(
					envRepo,
					blobStore,
					blobIdWithReadCloser{
						BlobId:     readCloser.BlobId,
						ReadCloser: out,
					},
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if err = utility.Wait(); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			},
		)
	}
}

func (cmd Cat) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStore := envBlobStore.GetDefaultBlobStore()

	blobWriter := cmd.makeBlobWriter(envBlobStore, blobStore)

	var blobStoreId blob_store_id.Id

	for _, arg := range req.PopArgs() {
		var blobId markl.Id

		if err := blobId.Set(arg); err == nil {
			if err := cmd.blob(blobStore, &blobId, blobWriter); err != nil {
				ui.Err().Print(err)
			}

			continue
		}

		if err := blobStoreId.Set(arg); err == nil {
			blobStore = envBlobStore.GetBlobStore(blobStoreId)
			blobWriter = cmd.makeBlobWriter(envBlobStore, blobStore)
			ui.Err().Printf("switched to blob store: %s", blobStoreId)
			continue
		}

		ui.Err().Print(
			errors.Errorf("invalid argument (not a blob id or store id): %s", arg),
		)
	}
}

func (cmd Cat) copy(
	envBlobStore env_repo.BlobStoreEnv,
	blobStore blob_stores.BlobStoreInitialized,
	readCloser blobIdWithReadCloser,
) (err error) {
	defer errors.DeferredCloser(&err, readCloser.ReadCloser)

	if cmd.PrefixSha {
		if _, err = delim_io.CopyWithPrefixOnDelim(
			'\n',
			readCloser.BlobId.String(),
			envBlobStore.GetUI(),
			readCloser.ReadCloser,
			true,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if _, err = io.Copy(
			envBlobStore.GetUIFile(),
			readCloser.ReadCloser,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (cmd Cat) blob(
	blobStore blob_stores.BlobStoreInitialized,
	blobId domain_interfaces.MarklId,
	blobWriter interfaces.FuncIter[blobIdWithReadCloser],
) (err error) {
	var reader domain_interfaces.BlobReader

	if reader, err = blobStore.MakeBlobReader(blobId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = blobWriter(blobIdWithReadCloser{BlobId: blobId, ReadCloser: reader}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

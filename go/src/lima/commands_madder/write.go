package commands_madder

import (
	"io"
	"sync/atomic"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/delta/script_value"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("write", &Write{})
}

type Write struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreLocal

	Check         bool
	UtilityBefore script_value.Utility
	UtilityAfter  script_value.Utility
}

var _ interfaces.CommandComponentWriter = (*Write)(nil)

func (cmd Write) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	// args := commandLine.FlagsOrArgs[1:]

	// if commandLine.InProgress != "" {
	// 	args = args[:len(args)-1]
	// }

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd *Write) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.Check,
		"check",
		false,
		"only check if the object already exists",
	)

	flagSet.Var(&cmd.UtilityBefore, "utility-before", "")
	flagSet.Var(&cmd.UtilityAfter, "utility-after", "")
}

type blobWriteResult struct {
	error
	domain_interfaces.MarklId
	Path string
}

// TODO add support for blob store ids
func (cmd Write) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStore := envBlobStore.GetDefaultBlobStore()

	var failCount atomic.Uint32
	var blobStoreId blob_store_id.Id

	sawStdin := false

	for _, arg := range req.PopArgs() {
		switch {
		case arg == "-" && sawStdin:
			ui.Err().Print("'-' passed in more than once. Ignoring")
			continue

		case arg == "-":
			sawStdin = true
		}

		result := blobWriteResult{Path: arg}

		var blobReader domain_interfaces.BlobReader

		{
			var err error

			if blobReader, err = env_dir.NewFileReaderOrErrNotExist(
				env_dir.DefaultConfig,
				arg,
			); errors.IsNotExist(err) {
				if err = blobStoreId.Set(arg); err != nil {
					req.Cancel(err)
					return
				}

				blobStore = envBlobStore.GetBlobStore(blobStoreId)
				ui.Debug().Printf("remote path: %q", blobStore.Config.Blob.(blob_store_configs.ConfigSFTPRemotePath).GetRemotePath())
				continue
			} else if err != nil {
				failCount.Add(1)
				result.error = err
				continue
			}
		}

		result.MarklId, result.error = cmd.doOne(blobStore, blobReader)

		if result.error != nil {
			envBlobStore.GetErr().Printf(
				"%s: (error: %q)",
				result.Path,
				result.error,
			)
			failCount.Add(1)
			continue
		}

		if result.IsNull() {
			ui.Err().Printf("digest for arg %q was null", arg)
			continue
		}

		hasBlob := blobStore.HasBlob(result.MarklId)

		if hasBlob {
			if cmd.Check {
				envBlobStore.GetUI().Printf(
					"%s %s (already checked in)",
					result.MarklId,
					result.Path,
				)
			} else {
				envBlobStore.GetUI().Printf(
					"%s %s (checked in)",
					result.MarklId,
					result.Path,
				)
			}
		} else {
			ui.Err().Printf(
				"%s %s (untracked)",
				result.MarklId,
				result.Path,
			)

			if cmd.Check {
				failCount.Add(1)
			}
		}
	}

	fc := failCount.Load()

	if fc > 0 {
		errors.ContextCancelWithBadRequestf(
			req,
			"untracked objects: %d",
			fc,
		)
		return
	}
}

// TODO rewrite to just return blobWriteResult
func (cmd Write) doOne(
	blobStore blob_stores.BlobStoreInitialized,
	blobReader domain_interfaces.BlobReader,
) (blobId domain_interfaces.MarklId, err error) {
	defer errors.DeferredCloser(&err, blobReader)

	var writeCloser domain_interfaces.BlobWriter

	if cmd.Check {
		{
			hash, hashRepool := blobStore.GetDefaultHashType().GetHash()
			var repool func()
			writeCloser, repool = markl_io.MakeWriterWithRepool(
				hash,
				nil,
			)
			defer func() { repool(); hashRepool() }()
		}
	} else {
		if writeCloser, err = blobStore.MakeBlobWriter(nil); err != nil {
			err = errors.Wrap(err)
			return blobId, err
		}
	}

	defer errors.DeferredCloser(&err, writeCloser)

	if _, err = io.Copy(writeCloser, blobReader); err != nil {
		err = errors.Wrap(err)
		return blobId, err
	}

	blobId = writeCloser.GetMarklId()

	return blobId, err
}

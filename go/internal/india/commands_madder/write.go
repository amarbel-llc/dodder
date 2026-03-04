package commands_madder

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/markl_io"
	"code.linenisgreat.com/dodder/go/internal/charlie/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/script_value"
	tap "github.com/amarbel-llc/purse-first/packages/tap-dancer/go"
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

func (cmd Write) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStore := envBlobStore.GetDefaultBlobStore()

	tw := tap.NewWriter(os.Stdout)

	var failCount atomic.Uint32
	var blobStoreId blob_store_id.Id

	sawStdin := false

	for _, arg := range req.PopArgs() {
		switch {
		case arg == "-" && sawStdin:
			tw.Comment("'-' passed in more than once. Ignoring")
			continue

		case arg == "-":
			sawStdin = true
		}

		result := blobWriteResult{Path: arg}

		resolved := command_components_madder.ResolveFileOrBlobStoreId(arg)

		if resolved.Err != nil {
			tw.NotOk(arg, tap_diagnostics.FromError(resolved.Err))
			failCount.Add(1)
			continue
		}

		if resolved.IsStoreSwitch {
			blobStoreId = resolved.BlobStoreId
			blobStore = envBlobStore.GetBlobStore(blobStoreId)
			tw.Comment(fmt.Sprintf("switched to blob store: %s", blobStoreId))
			continue
		}

		result.MarklId, result.error = cmd.doOne(blobStore, resolved.BlobReader)

		if result.error != nil {
			tw.NotOk(arg, tap_diagnostics.FromError(result.error))
			failCount.Add(1)
			continue
		}

		if result.IsNull() {
			tw.Skip(arg, "null digest")
			continue
		}

		hasBlob := blobStore.HasBlob(result.MarklId)

		if hasBlob {
			tw.Ok(fmt.Sprintf("%s %s", result.MarklId, result.Path))
		} else {
			if cmd.Check {
				tw.NotOk(
					fmt.Sprintf("%s %s", result.MarklId, result.Path),
					map[string]string{
						"severity": "fail",
						"message":  "untracked",
					},
				)
				failCount.Add(1)
			} else {
				tw.Ok(fmt.Sprintf("%s %s", result.MarklId, result.Path))
			}
		}
	}

	tw.Plan()

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

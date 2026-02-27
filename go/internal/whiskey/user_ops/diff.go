package user_ops

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/lib/bravo/comments"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/internal/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/store_fs"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

// TODO move to store_fs
type Diff struct {
	*local_working_copy.Repo

	object_metadata_fmt_triple_hyphen.FormatterFamily
}

func (op Diff) Run(
	remoteCheckedOut sku.SkuType,
	options object_metadata_fmt_triple_hyphen.FormatterOptions,
) (err error) {
	var localCheckedOut sku.SkuType

	{
		if localCheckedOut, err = op.GetEnvWorkspace().GetStoreFS().CheckoutOne(
			checkout_options.Options{
				CheckoutMode: checkout_mode.Make(checkout_mode.MetadataAndBlob),
				OptionsWithoutMode: checkout_options.OptionsWithoutMode{
					StoreSpecificOptions: store_fs.CheckoutOptions{
						Path: store_fs.PathOptionTempLocal,
					},
				},
			},
			remoteCheckedOut.GetSku(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.Deferred(&err, func() (err error) {
			if err = op.GetEnvWorkspace().GetStoreFS().DeleteCheckedOutInternal(
				localCheckedOut,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		})
	}

	wg := errors.MakeWaitGroupParallel()

	var mode checkout_mode.Mode

	local := localCheckedOut.GetSku()
	localContext := object_metadata_fmt_triple_hyphen.FormatterContext{
		EncoderContext:   local,
		FormatterOptions: options,
	}

	remote := remoteCheckedOut.GetSkuExternal()
	remoteCtx := object_metadata_fmt_triple_hyphen.FormatterContext{
		EncoderContext:   remote,
		FormatterOptions: options,
	}

	if mode, err = op.GetEnvWorkspace().GetStoreFS().GetCheckoutModeOrError(
		remote,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var rLeft, wLeft *os.File

	if rLeft, wLeft, err = os.Pipe(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var rRight, wRight *os.File

	if rRight, wRight, err = os.Pipe(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// sameTyp := il.GetTyp().Equals(el.GetTyp())
	internalInline := op.GetConfig().IsInlineType(local.GetType())
	externalInline := op.GetConfig().IsInlineType(remote.GetType())

	var fds *sku.FSItem

	if fds, err = op.GetEnvWorkspace().GetStoreFS().ReadFSItemFromExternal(remote); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var externalFD *fd.FD

	switch {
	case mode.IncludesMetadata():
		if internalInline && externalInline {
			wg.Do(op.makeDo(wLeft, op.InlineBlob, localContext))
			wg.Do(op.makeDo(wRight, op.InlineBlob, remoteCtx))
		} else {
			wg.Do(op.makeDo(wLeft, op.MetadataOnly, localContext))
			wg.Do(op.makeDo(wRight, op.MetadataOnly, remoteCtx))
		}

		externalFD = &fds.Object

	case internalInline && externalInline:
		wg.Do(
			op.makeDoBlob(
				wLeft,
				op.GetEnvRepo().GetDefaultBlobStore(),
				local.GetBlobDigest(),
			),
		)
		wg.Do(op.makeDoFD(wRight, &fds.Blob))
		externalFD = &fds.Blob

	default:
		wg.Do(op.makeDo(wLeft, op.MetadataOnly, localContext))
		wg.Do(op.makeDo(wRight, op.MetadataOnly, remoteCtx))
		externalFD = &fds.Blob
	}

	internalLabel := fmt.Sprintf(
		"%s:%s",
		local.GetObjectId(),
		strings.ToLower(local.GetGenre().String()),
	)

	externalLabel := op.GetEnvRepo().Rel(externalFD.GetPath())

	colorOptions := op.FormatColorOptionsOut(op.GetConfig().GetPrintOptions())
	colorString := "always"

	if colorOptions.OffEntirely {
		colorString = "never"
	}

	comments.Change("disambiguate internal and external, and object / blob")
	cmd := exec.Command(
		"diff",
		fmt.Sprintf("--color=%s", colorString),
		"-u",
		"--label", internalLabel,
		"--label", externalLabel,
		"/dev/fd/3",
		"/dev/fd/4",
	)

	cmd.ExtraFiles = []*os.File{rLeft, rRight}
	cmd.Stdout = op.GetOutFile()
	cmd.Stderr = op.GetErrFile()

	wg.Do(
		func() (err error) {
			defer errors.DeferredCloser(&err, rLeft)
			defer errors.DeferredCloser(&err, rRight)

			if err = cmd.Run(); err != nil {
				if cmd.ProcessState.ExitCode() == 1 {
					comments.Change("return non-zero exit code")
					err = nil
				} else {
					err = errors.Wrap(err)
				}

				return err
			}

			return err
		},
	)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (c Diff) makeDo(
	writeCloser io.WriteCloser,
	textFormatter object_metadata_fmt_triple_hyphen.Formatter,
	textFormatterContext object_metadata_fmt_triple_hyphen.FormatterContext,
) errors.FuncErr {
	return func() (err error) {
		defer errors.DeferredCloser(&err, writeCloser)

		if _, err = textFormatter.FormatMetadata(
			writeCloser,
			textFormatterContext,
		); err != nil {
			if errors.IsBrokenPipe(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}

		return err
	}
}

func (c Diff) makeDoBlob(
	w io.WriteCloser,
	arf domain_interfaces.BlobReaderFactory,
	sh domain_interfaces.MarklId,
) errors.FuncErr {
	return func() (err error) {
		defer errors.DeferredCloser(&err, w)

		var ar domain_interfaces.BlobReader

		if ar, err = arf.MakeBlobReader(sh); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, ar)

		if _, err = io.Copy(w, ar); err != nil {
			if errors.IsBrokenPipe(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}

		return err
	}
}

func (c Diff) makeDoFD(
	w io.WriteCloser,
	fd *fd.FD,
) errors.FuncErr {
	return func() (err error) {
		defer errors.DeferredCloser(&err, w)

		var f *os.File

		if f, err = files.OpenExclusiveReadOnly(fd.GetPath()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer errors.DeferredCloser(&err, f)

		if _, err = io.Copy(w, f); err != nil {
			if errors.IsBrokenPipe(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}

		return err
	}
}

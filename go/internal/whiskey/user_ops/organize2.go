package user_ops

import (
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/delta/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

type Organize2 struct {
	*local_working_copy.Repo
	organize_text.Metadata
}

func (op Organize2) Run(
	skus sku.CheckedOutMutableSet,
) (organizeResults organize_text.OrganizeResults, err error) {
	organizeResults.Original = skus

	organizeFlags := organize_text.MakeFlagsWithMetadata(op.Metadata)
	op.ApplyToOrganizeOptions(&organizeFlags.Options)
	organizeFlags.Skus = skus

	createOrganizeFileOp := CreateOrganizeFile{
		Repo: op.Repo,
		Options: op.Repo.MakeOrganizeOptionsWithOrganizeMetadata(
			organizeFlags,
			op.Metadata,
		),
	}

	var file *os.File

	fileExtensions := file_extensions.MakeDefaultConfig(op.GetConfig())

	organizeFileTemplate := fmt.Sprintf(
		"*.%s",
		fileExtensions.Organize,
	)

	if file, err = op.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
		organizeFileTemplate,
	); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	defer errors.DeferredCloser(&err, file)

	if organizeResults.Before, err = createOrganizeFileOp.RunAndWrite(
		file,
	); err != nil {
		err = errors.Wrap(err)
		return organizeResults, err
	}

	// TODO refactor into common vim processing loop
	for {
		openVimOp := OpenEditor{
			VimOptions: vim_cli_options_builder.New().
				WithFileType("dodder-organize").
				Build(),
		}

		if err = openVimOp.Run(op.Repo, file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		readOrganizeTextOp := ReadOrganizeFile{}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		if organizeResults.After, err = readOrganizeTextOp.Run(
			op.Repo,
			file,
			organize_text.NewMetadataWithOptionCommentLookup(
				organizeResults.Before.GetRepoId(),
				op.GetPrototypeOptionComments(),
			),
		); err != nil {
			if op.handleReadChangesError(op.Repo, err) {
				err = nil
				continue
			} else {
				ui.Err().Printf("aborting organize")
				return organizeResults, err
			}
		}

		break
	}

	return organizeResults, err
}

func (cmd Organize2) handleReadChangesError(
	envUI env_ui.Env,
	err error,
) (tryAgain bool) {
	var errorRead organize_text.ErrorRead

	if err != nil && !errors.As(err, &errorRead) {
		ui.Err().Printf("unrecoverable organize read failure: %s", err)
		tryAgain = false
		return tryAgain
	}

	return envUI.Retry(
		"reading changes failed",
		"edit and try again?",
		err,
	)
}

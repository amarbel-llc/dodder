package repo_actions

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/lib/0/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type Organize2 struct {
	*repo
	orgie.Metadata
}

func (op Organize2) Run(
	skus sku.CheckedOutMutableSet,
) (organizeResults orgie.OrganizeResults, err error) {
	organizeResults.Original = skus

	organizeFlags := orgie.MakeFlagsWithMetadata(op.Metadata)
	ApplyToOrganizeOptions(op.repo, &organizeFlags.Options)
	organizeFlags.Skus = skus

	createOrganizeFileOp := MakeCreateOrganizeFile(
		op.repo,
		MakeOrganizeOptionsWithOrganizeMetadata(
			op.repo,
			organizeFlags,
			op.Metadata,
		),
	)

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
		openVimOp := MakeOpenEditor(op.repo)
		openVimOp.VimOptions = vim_cli_options_builder.New().
			WithFileType("dodder-organize").
			Build()

		if err = openVimOp.Run(file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		// Reopen by path rather than seeking on the original handle: an
		// editor that saves via the common rename-over-original-path
		// idiom (e.g. vim's default backupcopy=auto) replaces the inode
		// at file.Name(), leaving the original *os.File pointing at the
		// old, unlinked-but-still-open inode. Seeking and reading from
		// that stale handle silently returns the pre-edit content.
		var reopened *os.File

		if reopened, err = os.Open(file.Name()); err != nil {
			err = errors.Wrap(err)
			return organizeResults, err
		}

		readOrganizeTextOp := MakeReadOrganizeFile(op.repo)

		organizeResults.After, err = readOrganizeTextOp.Run(
			reopened,
			orgie.NewMetadataWithSettingLookup(
				organizeResults.Before.GetRepoId(),
				op.GetPrototypeSettings(),
			),
		)

		errors.PanicIfError(reopened.Close())

		if err != nil {
			if op.handleReadChangesError(op.repo, err) {
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
	var errorRead orgie.ErrorRead

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

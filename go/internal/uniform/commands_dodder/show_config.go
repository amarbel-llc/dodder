package commands_dodder

import (
	"io"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("show-config", &ShowConfig{})
}

type ShowConfig struct {
	command_components_dodder.LocalWorkingCopy

	History bool
}

var (
	_ interfaces.CommandComponentWriter = (*ShowConfig)(nil)
	_ command.CommandWithArgs           = (*ShowConfig)(nil)
)

func (cmd *ShowConfig) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "digest",
			Description: "config blob digest to output (defaults to the head)",
		}},
	}}
}

func (cmd ShowConfig) GetDescription() command.Description {
	return command.Description{
		Short: "read the repository configuration log",
	}
}

func (cmd *ShowConfig) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopy.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.BoolVar(
		&cmd.History,
		"history",
		false,
		"print the config log history as box lines instead of a blob",
	)
}

func (cmd ShowConfig) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if cmd.History && len(args) > 0 {
		localWorkingCopy.Cancel(
			errors.ErrorWithStackf(
				"-history takes no positional arguments",
			),
		)
		return
	}

	log := config_log.Make(
		localWorkingCopy.GetEnvRepo(),
		command_components_dodder.InventoryLists{}.MakeInventoryListCoderCloset(
			localWorkingCopy.GetEnvRepo(),
		),
	)

	if cmd.History {
		cmd.runHistory(localWorkingCopy, log)
		return
	}

	cmd.runBlob(localWorkingCopy, log, args)
}

// runHistory prints every config log entry oldest->newest as a box line
// using the repo's archive box printer, producing the standard
// `[konfig @blake2b256-... <tai> !inventory_list-v2 ...]` form.
func (cmd ShowConfig) runHistory(
	localWorkingCopy *local_working_copy.Repo,
	log config_log.Log,
) {
	printer := localWorkingCopy.MakePrinterBoxArchive(
		localWorkingCopy.GetUIFile(),
		true,
	)

	for object, err := range log.All() {
		if err != nil {
			localWorkingCopy.Cancel(err)
			return
		}

		if err := printer(object); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}
}

// runBlob streams a single config blob's raw bytes to stdout. With no
// positional arg it resolves the config log head; with one arg it parses
// that arg as a markl id and streams the named blob directly.
func (cmd ShowConfig) runBlob(
	localWorkingCopy *local_working_copy.Repo,
	log config_log.Log,
	args []string,
) {
	var digest mad_domain_interfaces.MarklId

	if len(args) > 0 {
		var parsed markl.Id

		if err := parsed.Set(args[0]); err != nil {
			localWorkingCopy.Cancel(
				errors.Wrapf(err, "invalid markl id: %s", args[0]),
			)
			return
		}

		digest = &parsed
	} else {
		head, repoolHead, err := log.Head()
		if err != nil {
			// ErrEmpty is an expected clean exit (no config appended
			// yet), not a crash; Cancel surfaces it to the user the
			// same way sibling commands report empty state.
			localWorkingCopy.Cancel(err)
			return
		}

		defer repoolHead()

		digest = head.GetBlobDigest()
	}

	cmd.streamBlob(localWorkingCopy, digest)
}

// streamBlob copies the raw bytes of the blob named by digest from the
// read blob store to stdout.
func (cmd ShowConfig) streamBlob(
	localWorkingCopy *local_working_copy.Repo,
	digest mad_domain_interfaces.MarklId,
) {
	var readCloser io.ReadCloser

	{
		var err error

		if readCloser, err = localWorkingCopy.GetEnvRepo().
			GetReadBlobStore().
			MakeBlobReader(digest); err != nil {
			localWorkingCopy.Cancel(err)
			return
		}
	}

	defer errors.ContextMustClose(localWorkingCopy, readCloser)

	if _, err := ohio.CopyBuffered(
		localWorkingCopy.GetUIFile(),
		readCloser,
	); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}
}

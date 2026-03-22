package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("cat-object", &CatObject{
		Format: local_working_copy.FormatFlag{
			DefaultFormatter: local_working_copy.GetFormatFuncConstructorEntry(
				"log",
			),
		},
	})
}

type CatObject struct {
	command_components_dodder.LocalWorkingCopy

	Format local_working_copy.FormatFlag
}

var _ interfaces.CommandComponentWriter = (*CatObject)(nil)

func (cmd *CatObject) SetFlagDefinitions(flagDefs interfaces.CLIFlagDefinitions) {
	flagDefs.Var(
		&cmd.Format,
		"format",
		"format used when outputting objects to stdout",
	)
}

func (cmd CatObject) Run(req command.Request) {
	args := req.PopArgs()
	repo := cmd.MakeLocalWorkingCopy(req)

	if len(args) == 0 {
		errors.ContextCancelWithErrorf(
			repo,
			"expected one or more markl ID arguments",
		)
	}

	output := cmd.Format.MakeFormatFunc(
		repo,
		repo.GetUIFile(),
	)

	for _, arg := range args {
		id, repool := markl.GetId()

		if err := id.Set(arg); err != nil {
			repool()
			repo.Cancel(errors.Wrapf(err, "invalid markl id: %s", arg))
		}

		object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

		if !repo.GetStore().GetStreamIndex().ReadOneMarklId(id, object) {
			repool()
			repo.Cancel(errors.MakeErrNotFoundString(arg))
		}

		repool()

		if err := output(object); err != nil {
			repo.Cancel(err)
		}
	}
}

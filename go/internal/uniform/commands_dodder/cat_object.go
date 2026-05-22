package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	utility.AddCmd("cat-object", &CatObject{
		Format: local_working_copy.FormatFlag{
			DefaultFormatter: local_working_copy.GetFormatFuncConstructorEntry(
				"box",
			),
		},
	})
}

func (cmd CatObject) GetDescription() command.Description {
	return command.Description{
		Short: "output raw object content by markl id",
	}
}

type CatObject struct {
	command_components_dodder.LocalWorkingCopy

	Format local_working_copy.FormatFlag
}

var (
	_ interfaces.CommandComponentWriter = (*CatObject)(nil)
	_ command.CommandWithArgs           = (*CatObject)(nil)
)

func (cmd *CatObject) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "markl-ids",
			Description: "markl IDs of objects to output",
			Required:    true,
			Variadic:    true,
		}},
	}}
}

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

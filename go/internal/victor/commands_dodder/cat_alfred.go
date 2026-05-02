package commands_dodder

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/alfred_sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/alfred"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func init() {
	utility.AddCmd("cat-alfred", &CatAlfred{})
}

func (cmd CatAlfred) GetDescription() command.Description {
	return command.Description{
		Short: "output objects in Alfred workflow format",
	}
}

type CatAlfred struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	genres.Genre
}

var (
	_ interfaces.CommandComponentWriter = (*CatAlfred)(nil)
	_ command.CommandWithArgs           = (*CatAlfred)(nil)
)

func (cmd *CatAlfred) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd *CatAlfred) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagDefinitions)
	flagDefinitions.Var(
		&cmd.Genre,
		"genre",
		"extract this element from all matching objects",
	)
}

func (c CatAlfred) CompletionGenres() ids.Genre {
	return ids.MakeGenre(
		genres.Tag,
		genres.Type,
		genres.Zettel,
	)
}

func (cmd CatAlfred) Run(dep command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		dep,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultGenres(
				genres.Tag,
				genres.Type,
				genres.Zettel,
			),
		),
	)

	// this command does its own error handling
	wo := bufio.NewWriter(localWorkingCopy.GetUIFile())
	defer errors.ContextMustFlush(localWorkingCopy, wo)

	var aiw alfred.Writer

	itemPool := alfred.MakeItemPool()

	switch cmd.Genre {
	case genres.Type, genres.Tag:
		{
			var err error

			if aiw, err = alfred.NewDebouncingWriter(localWorkingCopy.GetUIFile()); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}

	default:
		{
			var err error

			if aiw, err = alfred.NewWriter(localWorkingCopy.GetUIFile(), itemPool); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}
	}

	var writer *alfred_sku.Writer

	{
		var err error

		if writer, err = alfred_sku.New(
			wo,
			localWorkingCopy.GetStore().GetAbbrStore().GetAbbr(),
			localWorkingCopy.SkuFormatBoxTransactedNoColor(),
			aiw,
			itemPool,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	defer errors.ContextMustClose(localWorkingCopy, writer)

	if err := localWorkingCopy.GetStore().QueryTransacted(
		queryGroup,
		func(object *sku.Transacted) (err error) {
			switch cmd.Genre {
			case genres.Tag:
				for tag := range object.GetMetadata().AllTags() {
					var tagObject *sku.Transacted

					if tagObject, err = localWorkingCopy.GetStore().ReadTransactedFromObjectId(
						tag,
					); err != nil {
						if errors.IsErrNotFound(err) {
							err = nil
							tagObject, tagObjectRepool := sku.GetTransactedPool().GetWithRepool()
							defer tagObjectRepool()

							if err = tagObject.GetObjectIdMutable().Set(tag.String()); err != nil {
								err = errors.Wrap(err)
								return err
							}
						} else {
							err = errors.Wrap(err)
							return err
						}
					}

					if err = writer.PrintOne(tagObject); err != nil {
						err = errors.Wrap(err)
						return err
					}
				}

			case genres.Type:
				typeLock := object.GetTypeLock()

				if typeLock.IsEmpty() {
					return err
				}

				if object, err = localWorkingCopy.GetStore().ReadTypeObject(
					typeLock,
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if err = writer.PrintOne(object); err != nil {
					err = errors.Wrap(err)
					return err
				}

			default:
				if err = writer.PrintOne(object); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			return err
		},
	); err != nil {
		writer.WriteError(err)
		err = nil
		return
	}
}

package commands_dodder

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/alfred"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/alfred_sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("cat-alfred", &CatAlfred{})
}

type CatAlfred struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	genres.Genre
}

var _ interfaces.CommandComponentWriter = (*CatAlfred)(nil)

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

							if err = tagObject.ObjectId.Set(tag.String()); err != nil {
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

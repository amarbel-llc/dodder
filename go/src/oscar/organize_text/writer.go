package organize_text

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/echo/format"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type writer struct {
	sku.ObjectFactory
	OmitLeadingEmptyLine bool
	objects.Metadata
	*format.LineWriter
	maxDepth int
	options  Options
}

func (av writer) write(a *Assignment) (err error) {
	spaceCount := av.maxDepth

	hinMaxWidth := 3

	if spaceCount < hinMaxWidth {
		spaceCount = hinMaxWidth
	}

	tab_prefix := strings.Repeat(" ", spaceCount+1)

	if a.GetDepth() == 0 && !av.OmitLeadingEmptyLine {
		av.WriteExactlyOneEmpty()
	} else if a.GetDepth() < 0 {
		err = errors.ErrorWithStackf("negative depth: %d", a.GetDepth())
		return err
	}

	if a.Transacted.GetMetadata().GetTags() != nil && a.Transacted.GetMetadata().GetTags().Len() > 0 {
		sharps := strings.Repeat("#", a.GetDepth())
		alignmentSpacing := strings.Repeat(" ", a.AlignmentSpacing())

		av.WriteLines(
			fmt.Sprintf(
				"%s%s %s%s",
				tab_prefix[len(sharps)-1:],
				sharps,
				alignmentSpacing,
				quiter.StringCommaSeparated(a.Transacted.GetMetadata().GetTags()),
			),
		)
		av.WriteExactlyOneEmpty()
	}

	write := func(object *obj) (err error) {
		var sb strings.Builder

		if object.tipe.IsDirectOrSelf() {
			sb.WriteString("- ")
		} else {
			sb.WriteString("% ")
		}

		cursor, _ := object.sku.Clone()
		cursorExternal := cursor.GetSkuExternal()
		cursorExternal.GetMetadataMutable().Subtract(av.Metadata)
		mes := ids.CloneTagSetMutable(cursorExternal.GetMetadataMutable().GetTags())

		if err = a.SubtractFromSet(mes); err != nil {
			err = errors.Wrap(err)
			return err
		}

		objects.SetTags(cursorExternal.GetMetadataMutable(), mes)

		if _, err = av.options.fmtBox.EncodeStringTo(
			cursor,
			&sb,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		av.WriteStringers(&sb)

		return err
	}

	a.Objects.Sort()

	for _, z := range a.Objects {
		if err = write(z); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if a.Objects.Len() > 0 {
		av.WriteExactlyOneEmpty()
	}

	for _, c := range a.Children {
		av.write(c)
	}

	av.WriteExactlyOneEmpty()

	return err
}

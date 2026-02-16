package alfred_sku

import (
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/charlie/expansion"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/alfred"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type Writer struct {
	alfredWriter alfred.Writer
	abbr         ids.Abbr
	organizeFmt  interfaces.StringEncoderTo[*sku.Transacted]
	alfred.ItemPool
}

func New(
	out io.Writer,
	abbr ids.Abbr,
	organizeFmt interfaces.StringEncoderTo[*sku.Transacted],
	aw alfred.Writer,
	itemPool alfred.ItemPool,
) (w *Writer, err error) {
	w = &Writer{
		abbr:         abbr,
		alfredWriter: aw,
		organizeFmt:  organizeFmt,
		ItemPool:     itemPool,
	}

	return w, err
}

func (writer *Writer) SetWriter(alfredWriter alfred.Writer) {
	writer.alfredWriter = alfredWriter
}

func (writer *Writer) PrintOne(object *sku.Transacted) (err error) {
	var item *alfred.Item
	g := object.GetGenre()

	switch g {
	case genres.Zettel:
		item = writer.zettelToItem(object)

	case genres.Tag:
		var tag ids.TagStruct

		if err = tag.Set(object.ObjectId.String()); err != nil {
			err = errors.Wrap(err)
			return err
		}

		item = writer.emitTag(object, &tag)

	default:
		item = writer.Get()
		item.Title = fmt.Sprintf("not implemented for genre: %q", g)
		item.Subtitle = sku.StringTaiGenreObjectIdObjectDigestBlobDigest(object)
	}

	writer.alfredWriter.WriteItem(item)

	return err
}

func (writer *Writer) WriteZettelId(e ids.ZettelId) (n int64, err error) {
	item := writer.zettelIdToItem(e)
	writer.alfredWriter.WriteItem(item)
	return n, err
}

func (writer *Writer) WriteError(in error) (n int64, out error) {
	if in == nil {
		return 0, nil
	}

	var errorGroup errors.Group

	if errors.As(in, &errorGroup) {
		for _, err := range errorGroup {
			item := writer.errorToItem(err)
			writer.alfredWriter.WriteItem(item)
		}
	} else {
		item := writer.errorToItem(in)
		writer.alfredWriter.WriteItem(item)
	}

	return n, out
}

func (writer Writer) Close() (err error) {
	return writer.alfredWriter.Close()
}

func (writer *Writer) addCommonMatches(
	object *sku.Transacted,
	item *alfred.Item,
) {
	id := &object.ObjectId

	matchBuilder, matchBuilderRepool := alfred.GetPoolMatchBuilder().GetWithRepool()
	defer matchBuilderRepool()

	{
		idString := id.String()
		matchBuilder.AddMatches(idString)
	}

	{
		seq := id.ToSeq()
		matchBuilder.AddMatchSeq(seq)
	}

	errors.PanicIfError(writer.abbr.AbbreviateZettelIdOnly(id))
	matchBuilder.AddMatches(id.String())

	{
		seq := id.ToSeq()
		matchBuilder.AddMatchSeq(seq)
	}

	matchBuilder.AddMatches(object.GetMetadataMutable().GetDescription().String())
	matchBuilder.AddMatches(object.GetType().String())

	for tag := range object.GetMetadata().AllTags() {
		for expandedTag := range expansion.ExpanderAll.Expand(tag.String()) {
			matchBuilder.AddMatches(expandedTag)
		}
	}

	{
		typeString := object.GetType().String()

		for expandedType := range expansion.ExpanderAll.Expand(typeString) {
			matchBuilder.AddMatches(expandedType)
		}
	}

	item.Match.Write(matchBuilder.Bytes())
	// a.Match.ReadFromBuffer(&mb.Buffer)
}

func (writer *Writer) zettelToItem(
	object *sku.Transacted,
) (item *alfred.Item) {
	item = writer.Get()

	item.Title = object.GetMetadata().GetDescription().String()

	es := quiter.StringCommaSeparated(
		object.GetMetadata().GetTags(),
	)

	k := &object.ObjectId
	ks := k.String()

	if item.Title == "" {
		item.Title = ks
		item.Subtitle = es
	} else {
		item.Subtitle = fmt.Sprintf("%s: %s %s", object.GetMetadata().GetType(), ks, es)
	}

	item.Arg = ks

	writer.addCommonMatches(object, item)

	item.Text.Copy = ks
	item.Uid = "dodder://" + ks

	{
		var sb strings.Builder

		if _, err := writer.organizeFmt.EncodeStringTo(object, &sb); err != nil {
			item = writer.errorToItem(err)
			return item
		}

		item.Mods["alt"] = alfred.Mod{
			Valid:    true,
			Arg:      sb.String(),
			Subtitle: sb.String(),
		}
	}

	return item
}

func (writer *Writer) emitTag(
	object *sku.Transacted,
	tag *ids.TagStruct,
) (item *alfred.Item) {
	item = writer.Get()

	item.Title = "@" + tag.String()

	item.Arg = tag.String()

	writer.addCommonMatches(object, item)

	item.Text.Copy = tag.String()
	item.Uid = "dodder://" + tag.String()

	return item
}

func (writer *Writer) errorToItem(err error) (a *alfred.Item) {
	a = writer.Get()

	a.Title = errors.Unwrap(err).Error()

	return a
}

func (writer *Writer) zettelIdToItem(e ids.ZettelId) (a *alfred.Item) {
	a = writer.Get()

	a.Title = e.String()

	a.Arg = e.String()

	mb, mbRepool := alfred.GetPoolMatchBuilder().GetWithRepool()
	defer mbRepool()

	mb.AddMatch(e.String())
	mb.AddMatch(e.GetHead())
	mb.AddMatch(e.GetTail())

	a.Match.ReadFromBuffer(&mb.Buffer)

	a.Text.Copy = e.String()
	a.Uid = "dodder://" + e.String()

	return a
}

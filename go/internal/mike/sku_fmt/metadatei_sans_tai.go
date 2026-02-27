package sku_fmt

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func StringMetadataSansTai(object *sku.Transacted) (str string) {
	sb := &strings.Builder{}

	sb.WriteString(object.GetGenre().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetObjectId().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetBlobDigest().String())

	m := object.GetMetadataMutable()

	t := m.GetType()

	if !t.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString(ids.FormattedString(m.GetType().ToType()))
	}

	es := m.GetTags()

	if es.Len() > 0 {
		sb.WriteString(" ")
		sb.WriteString(
			quiter.StringDelimiterSeparated(
				" ",
				m.GetTags(),
			),
		)
	}

	b := m.GetDescription()

	if !b.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString("\"" + b.String() + "\"")
	}

	return sb.String()
}

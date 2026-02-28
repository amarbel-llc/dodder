package objects

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
)

func StringSansTai(metadata *metadata) string {
	stringBuilder := &strings.Builder{}

	stringBuilder.WriteString(" ")
	stringBuilder.WriteString(metadata.DigBlob.String())

	tipe := metadata.GetType()

	if !tipe.IsEmpty() {
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(ids.FormattedString(metadata.GetType().ToType()))
	}

	tags := metadata.GetTags()

	if tags.Len() > 0 {
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(
			quiter.StringDelimiterSeparated[TagStruct](
				" ",
				metadata.GetTags(),
			),
		)
	}

	description := metadata.GetDescription()

	if !description.IsEmpty() {
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString("\"" + description.String() + "\"")
	}

	for field := range metadata.GetIndex().GetFields() {
		stringBuilder.WriteString(" ")
		fmt.Fprintf(stringBuilder, "%q=%q", field.Key, field.Value)
	}

	return stringBuilder.String()
}

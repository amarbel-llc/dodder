package sku

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
)

func String(object *Transacted) (str string) {
	return StringMetadataTaiMerkle(object)
}

func StringTaiGenreObjectIdObjectDigestBlobDigest(
	object *Transacted,
) (str string) {
	if object == nil {
		return "nil object!"
	}

	str = fmt.Sprintf(
		"%s %s %s %s %s",
		object.GetTai(),
		object.GetGenre(),
		object.GetObjectId(),
		object.GetObjectDigest(),
		object.GetBlobDigest(),
	)

	return str
}

func StringObjectIdBlobMetadataSansTai(object *Transacted) (str string) {
	if object == nil {
		return "nil object!"
	}

	str = fmt.Sprintf(
		"%s %s %s",
		object.GetObjectId(),
		object.GetBlobDigest(),
		StringMetadataSansTai(object),
	)

	return str
}

func StringMetadataTaiMerkle(object *Transacted) (str string) {
	if object == nil {
		return "nil object!"
	}

	tai := object.GetTai()
	taiFormatted := ids.MakeTaiRFC3339Value(tai)

	return fmt.Sprintf(
		"%s (%s) %s",
		tai,
		taiFormatted,
		StringMetadataSansTaiMerkle(object),
	)
}

func writeMarklIdWithFormatIfNecessary(
	stringBuilder *strings.Builder,
	id domain_interfaces.MarklId,
) {
	if id.IsNull() {
		return
	}

	stringBuilder.WriteString(" ")
	stringBuilder.WriteString(id.StringWithFormat())
}

func StringMetadataSansTai(object *Transacted) (str string) {
	sb := &strings.Builder{}

	sb.WriteString(object.GetGenre().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetObjectId().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetExternalObjectId().String())

	writeMarklIdWithFormatIfNecessary(sb, object.GetBlobDigest())

	metadata := object.GetMetadataMutable()

	t := metadata.GetType()

	if !t.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString(ids.FormattedString(metadata.GetType().ToType()))
	}

	es := metadata.GetTags()

	if es.Len() > 0 {
		sb.WriteString(" ")
		sb.WriteString(
			quiter.StringDelimiterSeparated(
				" ",
				metadata.GetTags(),
			),
		)
	}

	b := metadata.GetDescription()

	if !b.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString("\"" + b.String() + "\"")
	}

	for field := range metadata.GetIndex().GetFields() {
		sb.WriteString(" ")
		fmt.Fprintf(sb, "%q=%q", field.Key, field.Value)
	}

	return sb.String()
}

// TODO switch to using fmt.Fprintf for panic recovery
func StringMetadataSansTaiMerkle(object *Transacted) (str string) {
	sb := &strings.Builder{}

	sb.WriteString(object.GetGenre().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetObjectId().String())

	sb.WriteString(" ")
	sb.WriteString(object.GetExternalObjectId().String())

	writeMarklIdWithFormatIfNecessary(sb, object.GetMetadata().GetRepoPubKey())
	writeMarklIdWithFormatIfNecessary(sb, object.GetMetadata().GetObjectSig())
	writeMarklIdWithFormatIfNecessary(sb, object.GetMetadata().GetMotherObjectSig())
	writeMarklIdWithFormatIfNecessary(sb, object.GetObjectDigest())
	writeMarklIdWithFormatIfNecessary(sb, object.GetBlobDigest())

	metadata := object.GetMetadataMutable()

	t := metadata.GetType()

	if !t.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString(ids.FormattedString(metadata.GetType().ToType()))
		sb.WriteString("@")
		sb.WriteString(metadata.GetTypeLock().GetValue().String())
	}

	es := metadata.GetTags()

	if es.Len() > 0 {
		sb.WriteString(" ")
		sb.WriteString(
			quiter.StringDelimiterSeparated(
				" ",
				metadata.GetTags(),
			),
		)
	}

	b := metadata.GetDescription()

	if !b.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString("\"" + b.String() + "\"")
	}

	for field := range metadata.GetIndex().GetFields() {
		sb.WriteString(" ")
		fmt.Fprintf(sb, "%q=%q", field.Key, field.Value)
	}

	return sb.String()
}

func StringMetadataSansTaiMerkle2(
	object objects.MetadataMutable,
) (str string) {
	sb := &strings.Builder{}

	writeMarklIdWithFormatIfNecessary(sb, object.GetRepoPubKey())
	writeMarklIdWithFormatIfNecessary(sb, object.GetObjectSig())
	writeMarklIdWithFormatIfNecessary(sb, object.GetObjectDigest())
	writeMarklIdWithFormatIfNecessary(sb, object.GetBlobDigest())

	metadata := object.GetMetadataMutable()

	tipe := metadata.GetType()

	if !tipe.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString(ids.FormattedString(metadata.GetType().ToType()))
	}

	es := metadata.GetTags()

	if es.Len() > 0 {
		sb.WriteString(" ")
		sb.WriteString(
			quiter.StringDelimiterSeparated(
				" ",
				metadata.GetTags(),
			),
		)
	}

	b := metadata.GetDescription()

	if !b.IsEmpty() {
		sb.WriteString(" ")
		sb.WriteString("\"" + b.String() + "\"")
	}

	for field := range metadata.GetIndex().GetFields() {
		sb.WriteString(" ")
		fmt.Fprintf(sb, "%q=%q", field.Key, field.Value)
	}

	return sb.String()
}

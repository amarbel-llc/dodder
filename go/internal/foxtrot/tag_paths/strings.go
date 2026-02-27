package tag_paths

import "strings"

type StringMarshalerForward Path

func (stringMarshaler *StringMarshalerForward) String() string {
	var stringBuilder strings.Builder

	stringBuilder.WriteString("[")

	afterFirst := false
	count := (*Path)(stringMarshaler).Len()

	for index := count - 1; index >= 0; index-- {
		if afterFirst {
			stringBuilder.WriteString(" -> ")
		}

		afterFirst = true

		tag := (*stringMarshaler)[index]
		stringBuilder.Write(tag.Bytes())
	}

	stringBuilder.WriteString("]")

	return stringBuilder.String()
}

type StringMarshalerBackward Path

func (stringMarshaler *StringMarshalerBackward) String() string {
	var stringBuilder strings.Builder

	stringBuilder.WriteString("[")

	afterFirst := false
	count := (*Path)(stringMarshaler).Len()

	for index := range count {
		if afterFirst {
			stringBuilder.WriteString(" -> ")
		}

		afterFirst = true

		tag := (*stringMarshaler)[index]
		stringBuilder.Write(tag.Bytes())
	}

	stringBuilder.WriteString("]")

	return stringBuilder.String()
}

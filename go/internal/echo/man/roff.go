package man

import (
	"fmt"
	"strings"
)

func roffHeader(name string, section int, date, source, manual string) string {
	return fmt.Sprintf(
		".TH %s %d %q %q %q",
		strings.ToUpper(name),
		section,
		date,
		source,
		manual,
	)
}

func roffSection(name string) string {
	return fmt.Sprintf(".SH %s", strings.ToUpper(name))
}

func roffBold(s string) string {
	return fmt.Sprintf("\\fB%s\\fR", s)
}

func roffItalic(s string) string {
	return fmt.Sprintf("\\fI%s\\fR", s)
}

func roffEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\e`)
	s = strings.ReplaceAll(s, "-", `\-`)

	var b strings.Builder

	for i, line := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteByte('\n')
		}

		if strings.HasPrefix(line, ".") {
			b.WriteString(`\&`)
		}

		b.WriteString(line)
	}

	return b.String()
}

func roffParagraph() string {
	return ".PP"
}

func roffTaggedParagraph() string {
	return ".TP"
}

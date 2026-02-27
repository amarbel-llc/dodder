package object_fmt_digest

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type UniqueObject struct {
	DuplicateCount int
	Object         FormatterContext
}

type CLIFlag struct {
	DuplicateObjectDigestFormats []string
	Duplicates                   map[string]int
}

func (flag *CLIFlag) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	flag.Duplicates = make(map[string]int)

	flagSet.Func("dup-object-digest_format", "", func(value string) (err error) {
		return
	})
}

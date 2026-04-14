package object_fmt_digest

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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

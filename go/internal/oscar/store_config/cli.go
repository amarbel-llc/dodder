package store_config

import "fmt"

func (config Config) GetCLIFlags() (flags []string) {
	printOptions := config.GetPrintOptions()

	flags = append(
		flags,
		fmt.Sprintf("-print-time=%t", printOptions.BoxPrintTime),
		fmt.Sprintf("-print-colors=%t", printOptions.PrintColors),
		fmt.Sprintf("-abbreviate-shas=%t", printOptions.AbbreviateMarklIds),
		fmt.Sprintf(
			"-abbreviate-zettel-ids=%t",
			printOptions.AbbreviateZettelIds,
		),
		fmt.Sprintf("-print-flush=%t", printOptions.PrintFlush),
	)

	if config.Verbose {
		flags = append(flags, "-verbose")
	}

	return flags
}

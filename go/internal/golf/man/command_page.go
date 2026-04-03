package man

import (
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
)

func generateCommandPage(
	w io.Writer,
	config PageConfig,
	name string,
	cmd command.Cmd,
) error {
	pageName := fmt.Sprintf("%s-%s", config.BinaryName, name)
	synopsisCmd := fmt.Sprintf("%s %s", config.BinaryName, name)

	manual := fmt.Sprintf("%s Manual", config.Source)

	fmt.Fprintln(w, roffHeader(
		roffEscape(pageName),
		config.Section,
		config.Date,
		config.Version,
		manual,
	))

	// NAME
	fmt.Fprintln(w, roffSection("NAME"))

	var description command.Description

	if withDesc, ok := cmd.(command.CommandWithDescription); ok {
		description = withDesc.GetDescription()
	}

	if description.Short != "" {
		fmt.Fprintf(
			w,
			"%s \\- %s\n",
			roffEscape(pageName),
			roffEscape(description.Short),
		)
	} else {
		fmt.Fprintln(w, roffEscape(pageName))
	}

	// SYNOPSIS
	fmt.Fprintln(w, roffSection("SYNOPSIS"))
	fmt.Fprintf(
		w,
		"%s [%s] [%s]\n",
		roffBold(roffEscape(synopsisCmd)),
		roffItalic("options"),
		roffItalic("args..."),
	)

	// DESCRIPTION
	if description.Long != "" {
		fmt.Fprintln(w, roffSection("DESCRIPTION"))
		fmt.Fprintln(w, roffEscape(description.Long))
	} else if description.Short != "" {
		fmt.Fprintln(w, roffSection("DESCRIPTION"))
		fmt.Fprintln(w, roffEscape(description.Short))
	}

	// OPTIONS
	flagSet := flags.NewFlagSet(name, flags.ContinueOnError)

	if writer, ok := cmd.(interfaces.CommandComponentWriter); ok {
		writer.SetFlagDefinitions(flagSet)
	}

	if flagCount := countFlags(flagSet); flagCount > 0 {
		fmt.Fprintln(w, roffSection("OPTIONS"))
		writeFlags(w, flagSet)
	}

	// SEE ALSO
	fmt.Fprintln(w, roffSection("SEE ALSO"))
	fmt.Fprintf(
		w,
		"%s\n",
		roffBold(roffEscape(fmt.Sprintf("%s(1)", config.BinaryName))),
	)

	return nil
}

func countFlags(flagSet *flags.FlagSet) int {
	count := 0

	flagSet.VisitAll(func(f *flags.Flag) {
		count++
	})

	return count
}

func writeFlags(w io.Writer, flagSet *flags.FlagSet) {
	flagSet.VisitAll(func(f *flags.Flag) {
		fmt.Fprintln(w, roffTaggedParagraph())

		flagName := roffBold(roffEscape(fmt.Sprintf("-%s", f.Name)))

		if f.DefValue != "" && f.DefValue != "false" {
			fmt.Fprintf(
				w,
				"%s %s\n",
				flagName,
				roffItalic("value"),
			)
		} else {
			fmt.Fprintln(w, flagName)
		}

		usage := strings.TrimSpace(f.Usage)
		if usage != "" {
			fmt.Fprintln(w, roffEscape(usage))
		}

		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			fmt.Fprintf(w, "Default: %s\n", roffEscape(f.DefValue))
		}
	})
}

package man

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/flags"
)

func generateUtilityPage(
	w io.Writer,
	config PageConfig,
	utility command.Utility,
) error {
	manual := fmt.Sprintf("%s Manual", config.Source)

	fmt.Fprintln(w, roffHeader(
		strings.ToUpper(config.BinaryName),
		config.Section,
		config.Date,
		config.Version,
		manual,
	))

	// NAME
	fmt.Fprintln(w, roffSection("NAME"))
	fmt.Fprintf(
		w,
		"%s \\- %s\n",
		roffEscape(config.BinaryName),
		roffEscape(config.Description),
	)

	// SYNOPSIS
	fmt.Fprintln(w, roffSection("SYNOPSIS"))
	fmt.Fprintf(
		w,
		"%s [%s] %s [%s] [%s]\n",
		roffBold(roffEscape(config.BinaryName)),
		roffItalic("global\\-options"),
		roffItalic("command"),
		roffItalic("command\\-options"),
		roffItalic("args..."),
	)

	// DESCRIPTION
	if config.LongDescription != "" {
		fmt.Fprintln(w, roffSection("DESCRIPTION"))
		fmt.Fprintln(w, roffEscape(config.LongDescription))
	}

	// GLOBAL OPTIONS
	globalFlagSet := flags.NewFlagSet("global", flags.ContinueOnError)

	if writer, ok := utility.GetConfigAny().(interfaces.CommandComponentWriter); ok {
		writer.SetFlagDefinitions(globalFlagSet)
	}

	if flagCount := countFlags(globalFlagSet); flagCount > 0 {
		fmt.Fprintln(w, roffSection("GLOBAL OPTIONS"))
		writeFlags(w, globalFlagSet)
	}

	// COMMANDS
	type cmdEntry struct {
		name        string
		description string
	}

	var entries []cmdEntry

	for name, cmd := range utility.AllCmds() {
		var short string

		if withDesc, ok := cmd.(command.CommandWithDescription); ok {
			short = withDesc.GetDescription().Short
		}

		entries = append(entries, cmdEntry{name: name, description: short})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	fmt.Fprintln(w, roffSection("COMMANDS"))

	for _, entry := range entries {
		fmt.Fprintln(w, roffTaggedParagraph())

		if entry.description != "" {
			fmt.Fprintf(
				w,
				"%s\n%s\n",
				roffBold(roffEscape(entry.name)),
				roffEscape(entry.description),
			)
		} else {
			fmt.Fprintln(w, roffBold(roffEscape(entry.name)))
		}
	}

	// SEE ALSO
	fmt.Fprintln(w, roffSection("SEE ALSO"))

	var seeAlso []string

	for _, entry := range entries {
		ref := fmt.Sprintf(
			"%s",
			roffBold(roffEscape(
				fmt.Sprintf("%s-%s(1)", config.BinaryName, entry.name),
			)),
		)

		seeAlso = append(seeAlso, ref)
	}

	fmt.Fprintln(w, strings.Join(seeAlso, ",\n"))

	return nil
}

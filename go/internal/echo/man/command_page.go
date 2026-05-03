package man

import (
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/alfa/flags"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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

	// Collect flags and args metadata for SYNOPSIS, OPTIONS, and ARGUMENTS.
	flagSet := flags.NewFlagSet(name, flags.ContinueOnError)

	if writer, ok := cmd.(interfaces.CommandComponentWriter); ok {
		writer.SetFlagDefinitions(flagSet)
	}

	var argGroups []command.ArgGroup
	hasArgMetadata := false

	if argsCmd, ok := cmd.(command.CommandWithArgs); ok {
		hasArgMetadata = true
		argGroups = argsCmd.GetArgs()
	}

	// SYNOPSIS
	fmt.Fprintln(w, roffSection("SYNOPSIS"))

	fmt.Fprintf(w, "%s", roffBold(roffEscape(synopsisCmd)))

	if countFlags(flagSet) > 0 {
		fmt.Fprintf(w, " [%s]", roffItalic("options"))
	}

	if len(argGroups) > 0 {
		writeSynopsisArgs(w, argGroups)
	} else if !hasArgMetadata {
		fmt.Fprintf(w, " [%s]", roffItalic("args..."))
	}

	fmt.Fprintln(w)

	// DESCRIPTION
	if description.Long != "" {
		fmt.Fprintln(w, roffSection("DESCRIPTION"))
		fmt.Fprintln(w, roffEscape(description.Long))
	} else if description.Short != "" {
		fmt.Fprintln(w, roffSection("DESCRIPTION"))
		fmt.Fprintln(w, roffEscape(description.Short))
	}

	// OPTIONS
	if countFlags(flagSet) > 0 {
		fmt.Fprintln(w, roffSection("OPTIONS"))
		writeFlags(w, flagSet)
	}

	// ARGUMENTS
	if len(argGroups) > 0 {
		writeArgsSection(w, argGroups)
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

// writeSynopsisArgs appends arg placeholders to the SYNOPSIS line.
// Required args use <name>, optional use [name], variadic append "...".
func writeSynopsisArgs(w io.Writer, groups []command.ArgGroup) {
	for _, group := range groups {
		for _, arg := range group.Args {
			name := arg.Name

			if arg.Variadic {
				name += "..."
			}

			if arg.Required {
				fmt.Fprintf(w, " <%s>", roffItalic(roffEscape(name)))
			} else {
				fmt.Fprintf(w, " [%s]", roffItalic(roffEscape(name)))
			}
		}
	}
}

// writeArgsSection writes the ARGUMENTS section with per-arg descriptions.
func writeArgsSection(w io.Writer, groups []command.ArgGroup) {
	hasContent := false

	for _, group := range groups {
		for _, arg := range group.Args {
			if arg.Description == "" && group.Description == "" {
				continue
			}

			hasContent = true

			break
		}
	}

	if !hasContent {
		return
	}

	fmt.Fprintln(w, roffSection("ARGUMENTS"))

	for _, group := range groups {
		if group.Description != "" && group.Name != "" {
			fmt.Fprintln(w, roffParagraph())
			fmt.Fprintf(w, "%s:\n", roffBold(roffEscape(group.Name)))
			fmt.Fprintln(w, roffEscape(group.Description))
		}

		for _, arg := range group.Args {
			fmt.Fprintln(w, roffTaggedParagraph())

			name := arg.Name

			if arg.Variadic {
				name += "..."
			}

			fmt.Fprintln(w, roffBold(roffEscape(name)))

			desc := arg.Description
			if desc == "" {
				desc = group.Description
			}

			if desc != "" {
				fmt.Fprintln(w, roffEscape(desc))
			}

			if len(arg.EnumValues) > 0 {
				fmt.Fprintf(
					w,
					"Values: %s\n",
					roffEscape(strings.Join(arg.EnumValues, ", ")),
				)
			}

			if arg.Value != nil {
				if defVal := arg.Value.String(); defVal != "" {
					fmt.Fprintf(w, "Default: %s\n", roffEscape(defVal))
				}
			}
		}
	}
}

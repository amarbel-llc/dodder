package commands_dodder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
)

// versionCmd is the singleton registered with the utility. SetVersion
// mutates its fields in place — each cmd/* main calls SetVersion before
// utility.Run, so no synchronization is needed.
var versionCmd = &Version{
	version: "dev",
	commit:  "unknown",
}

func init() {
	utility.AddCmd("version", versionCmd)
}

// SetVersion populates the version subcommand with the ldflag-injected
// values from `package main`. Each cmd/* main is responsible for calling
// this before utility.Run.
func SetVersion(v, c string) {
	versionCmd.version = v
	versionCmd.commit = c
}

// dodderVersionString returns the ldflag-injected build identity in the same
// "<version>+<commit>" form the version subcommand prints. Used to surface
// which binary is serving the MCP (so an MCP-observed bug report carries the
// version that produced it).
func dodderVersionString() string {
	return versionCmd.version + "+" + versionCmd.commit
}

type Version struct {
	version string
	commit  string
}

func (cmd Version) GetDescription() command.Description {
	return command.Description{
		Short: "print dodder build version and commit",
	}
}

func (cmd Version) Run(req command.Request) {
	fmt.Fprintln(os.Stdout, cmd.version+"+"+cmd.commit)
}

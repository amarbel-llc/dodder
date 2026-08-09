package commands_dodder

import (
	"fmt"
	"os"
	"time"

	"code.linenisgreat.com/dodder/go/internal/bravo/registry"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	tap "code.linenisgreat.com/tap/go/pkgs/writer"
)

func init() {
	utility.AddCmd("registry-gc", &RegistryGC{
		Retention: registry.DefaultRetention().String(),
	})
}

// RegistryGC prunes dangling entries from the per-host repo registry index
// (RFC-0007 registry v1; the dodder twin of `madder registry-gc`). A
// dangling entry is a symlink whose target config-seed no longer exists —
// the repo was deleted or moved out from under the index. Entries younger
// than -retention (measured from registration time) are kept as a grace
// window against transient unavailability. v1 keeps no tombstone: a pruned
// entry leaves nothing behind.
type RegistryGC struct {
	// Retention is a Go duration string (e.g. "720h", "30m"). "0" is a
	// no-op.
	Retention string
}

var (
	_ interfaces.CommandComponentWriter = (*RegistryGC)(nil)
	_ command.CommandWithArgs           = (*RegistryGC)(nil)
)

func (cmd RegistryGC) GetDescription() command.Description {
	return command.Description{
		Short: "prune stale entries from the per-host repo registry",
		Long: "Remove dangling entries from the per-host registry index at " +
			"$XDG_STATE_HOME/dodder/index — symlinks whose target config-seed " +
			"no longer exists because the repo was deleted or moved. Only " +
			"entries older than -retention (a Go duration, default 720h / 30 " +
			"days, measured from registration time) are pruned; younger " +
			"dangling entries are kept as a grace window against a " +
			"transiently-unavailable repo (e.g. an unmounted filesystem). " +
			"Live entries are never touched. -retention=0 is a no-op. The " +
			"index is advisory: it feeds `dodder repos-list` only, never " +
			"repo resolution.",
	}
}

func (cmd *RegistryGC) GetArgs() []command.ArgGroup { return nil }

func (cmd *RegistryGC) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(
		&cmd.Retention,
		"retention",
		cmd.Retention,
		"prune dangling entries older than this Go duration (e.g. \"720h\"); "+
			"\"0\" is a no-op",
	)
}

func (cmd RegistryGC) Run(req command.Request) {
	req.AssertNoMoreArgs()

	retention, err := time.ParseDuration(cmd.Retention)
	if err != nil {
		errors.ContextCancelWithBadRequestf(
			req,
			"invalid -retention %q: %s",
			cmd.Retention,
			err,
		)
		return
	}

	removed, err := registry.GC(retention)
	if err != nil {
		req.Cancel(err)
		return
	}

	tw := tap.NewWriter(os.Stdout)
	tw.Ok(fmt.Sprintf(
		"registry-gc pruned %d stale %s (retention %s)",
		removed,
		pluralRegistryEntries(removed),
		retention,
	))
	tw.Plan()
}

func pluralRegistryEntries(n int) string {
	if n == 1 {
		return "entry"
	}
	return "entries"
}

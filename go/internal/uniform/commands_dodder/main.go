package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"

	// Blank import: triggers init() in dodder's own markl_registrations
	// package, which calls markl.RegisterPurpose for every dodder-*
	// purpose (Repo*, Object*, Blob*, Request*) and chains madder's
	// madder-only registrations (madder-* purposes + the dodder/zit
	// private-key aliases). Without this, any dodder code path that
	// calls into piggy's markl panics with
	//     no purpose registered for id "..."
	// Triggered first via blob_store_configs encryption flag parsing,
	// then signature verification, then anywhere markl.Id.Set parses a
	// purpose-tagged blech32 string. dodder #144 dropped its local
	// bravo/markl/purposes.go in favor of madder's published
	// markl_registrations; the madder#255 ownership cutover moved the
	// dodder-* vocabulary into dodder's own package.
	_ "code.linenisgreat.com/dodder/go/internal/0/markl_registrations"
)

var utility = command.MakeUtility(
	"dodder",
	repo_config_cli.Default(),
)

func GetUtility(name string) command.Utility {
	return utility
}

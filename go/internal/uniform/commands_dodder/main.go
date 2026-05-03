package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"

	// Blank import: triggers init() in madder's markl_registrations
	// package, which calls markl.RegisterPurpose for every purpose dodder
	// uses (Repo*, Object*, Blob*, Madder*, Request*) plus the
	// dodder/zit private-key alias. Without this, any dodder code path
	// that calls into madder's markl panics with
	//     no purpose registered for id "..."
	// Triggered first via blob_store_configs encryption flag parsing,
	// then signature verification, then anywhere madker.Id.Set parses a
	// purpose-tagged blech32 string. dodder #144 dropped its local
	// bravo/markl/purposes.go in favor of madder's published
	// markl_registrations; this import is what makes that swap effective.
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
)

var utility = command.MakeUtility(
	"dodder",
	repo_config_cli.Default(),
)

func GetUtility(name string) command.Utility {
	return utility
}

package domain_interfaces

import (
	madder_di "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	ConfigDryRunGetter    = madder_di.ConfigDryRunGetter
	ConfigDryRunSetter    = madder_di.ConfigDryRunSetter
	MutableConfigDryRun   = madder_di.MutableConfigDryRun
	Config                = madder_di.Config
	MutableConfig         = madder_di.MutableConfig
	CLIConfigProvider     = madder_di.CLIConfigProvider
	RepoCLIConfigProvider = madder_di.RepoCLIConfigProvider
)

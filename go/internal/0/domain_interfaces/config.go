package domain_interfaces

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	ConfigDryRunGetter    = mad_domain_interfaces.ConfigDryRunGetter
	ConfigDryRunSetter    = mad_domain_interfaces.ConfigDryRunSetter
	MutableConfigDryRun   = mad_domain_interfaces.MutableConfigDryRun
	Config                = mad_domain_interfaces.Config
	MutableConfig         = mad_domain_interfaces.MutableConfig
	CLIConfigProvider     = mad_domain_interfaces.CLIConfigProvider
	RepoCLIConfigProvider = mad_domain_interfaces.RepoCLIConfigProvider
)

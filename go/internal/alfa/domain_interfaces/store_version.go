package domain_interfaces

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

// TODO combine with config_immutable.StoreVersion and make a sealed struct
type StoreVersion interface {
	interfaces.Stringer
	GetInt() int
}

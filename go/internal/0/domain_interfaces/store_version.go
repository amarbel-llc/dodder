package domain_interfaces

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

// TODO combine with config_immutable.StoreVersion and make a sealed struct
type StoreVersion interface {
	interfaces.Stringer
	GetInt() int
}

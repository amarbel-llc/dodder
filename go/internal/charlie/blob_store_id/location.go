package blob_store_id

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type (
	LocationType interface {
		LocationTypeGetter
		location()
		GetPrefix() rune
	}

	LocationTypeGetter interface {
		GetLocationType() LocationType
	}

	//go:generate stringer -type=location
	location int
)

const (
	LocationTypeUnknown = location(iota)
	LocationTypeCwd
	LocationTypeXDGUser
	LocationTypeXDGSystem
)

var (
	_ LocationTypeGetter = location(0)
	_ LocationType       = location(0)
)

func (location) location() {}

func (location location) GetLocationType() LocationType { return location }

func (location *location) SetPrefix(firstChar rune) (err error) {
	switch firstChar {
	case '/':
		*location = LocationTypeXDGSystem

	case '~':
		*location = LocationTypeXDGUser

	case '.':
		*location = LocationTypeCwd

	case '_':
		*location = LocationTypeUnknown

	default:
		err = errors.Errorf(
			"unsupported rune for location type: %q",
			string(firstChar),
		)

		return err
	}

	return err
}

func (location location) IsPrefix(r rune) bool {
	switch r {
	case '/', '~', '.', '_':
		return true

	default:
		return false
	}
}

func (location location) GetPrefix() rune {
	switch location {
	case LocationTypeXDGSystem:
		return '/'

	case LocationTypeXDGUser:
		return 0

	case LocationTypeCwd:
		return '.'

	case LocationTypeUnknown:
		return '_'

	default:
		panic(errors.Errorf("unsupported location type: %q", location))
	}
}

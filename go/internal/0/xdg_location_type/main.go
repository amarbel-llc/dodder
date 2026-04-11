package xdg_location_type

import "code.linenisgreat.com/dodder/go/lib/bravo/errors"

type (
	Type interface {
		TypeGetter
		xdgLocationType()
		GetPrefix() rune
	}

	TypeGetter interface {
		GetLocationType() Type
	}

	//go:generate stringer -type=Typee
	Typee int
)

const (
	Unknown = Typee(iota)
	Cwd
	XDGUser
	XDGSystem
)

var (
	_ TypeGetter = Typee(0)
	_ Type       = Typee(0)
)

func (Typee) xdgLocationType() {}

func (t Typee) GetLocationType() Type { return t }

func (t *Typee) SetPrefix(firstChar rune) (err error) {
	switch firstChar {
	case '/':
		*t = XDGSystem

	case '~':
		*t = XDGUser

	case '.':
		*t = Cwd

	case '_':
		*t = Unknown

	default:
		err = errors.Errorf(
			"unsupported rune for location type: %q",
			string(firstChar),
		)

		return err
	}

	return err
}

func (t Typee) IsPrefix(r rune) bool {
	switch r {
	case '/', '~', '.', '_':
		return true

	default:
		return false
	}
}

func (t Typee) GetPrefix() rune {
	switch t {
	case XDGSystem:
		return '/'

	case XDGUser:
		return 0

	case Cwd:
		return '.'

	case Unknown:
		return '_'

	default:
		panic(errors.Errorf("unsupported location type: %q", t))
	}
}

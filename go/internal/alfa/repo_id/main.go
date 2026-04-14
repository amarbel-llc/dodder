package repo_id

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/madder/go/pkgs/xdg_location_type"
)

type Id struct {
	locationType xdg_location_type.Typee
	isSet        bool
}

func (id Id) IsEmpty() bool {
	return !id.isSet
}

func (id Id) GetLocationType() xdg_location_type.Type {
	return id.locationType
}

func (id *Id) Set(value string) (err error) {
	switch value {
	case "":
		id.isSet = false
		return nil

	case ".":
		id.locationType = xdg_location_type.Cwd
		id.isSet = true
		return nil

	case "/":
		id.locationType = xdg_location_type.XDGSystem
		id.isSet = true
		return nil

	default:
		if len(value) > 1 && value[0] == '/' {
			err = errors.Errorf(
				"remote repo selection (/%s) not yet implemented",
				value[1:],
			)
			return err
		}

		err = errors.Errorf("invalid repo_id: %q (expected . or /)", value)
		return err
	}
}

func (id Id) String() string {
	if !id.isSet {
		return ""
	}

	prefix := id.locationType.GetPrefix()
	if prefix == 0 {
		return ""
	}

	return string(prefix)
}

func (id Id) IsCwd() bool {
	return id.isSet && id.locationType == xdg_location_type.Cwd
}

func (id Id) IsSystem() bool {
	return id.isSet && id.locationType == xdg_location_type.XDGSystem
}

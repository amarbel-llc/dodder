package blob_store_id

import (
	"encoding"
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type Id struct {
	location location
	id       string
}

var (
	_ interfaces.Stringer      = Id{}
	_ interfaces.Setter        = &Id{}
	_ encoding.TextMarshaler   = Id{}
	_ encoding.TextUnmarshaler = &Id{}
)

func Make(id string) Id {
	return Id{
		location: LocationTypeXDGUser,
		id:       id,
	}
}

func MakeWithLocation(id string, locationType LocationTypeGetter) Id {
	return Id{
		location: locationType.GetLocationType().(location),
		id:       id,
	}
}

func (id Id) GetName() string {
	return id.id
}

func (id Id) IsEmpty() bool {
	return id.id == ""
}

func (id Id) GetLocationType() LocationType {
	return id.location
}

func (id Id) String() string {
	if id.id == "" {
		return ""
	}

	prefix := id.location.GetPrefix()

	if prefix == 0 {
		return id.id
	}

	return fmt.Sprintf("%c%s", prefix, id.id)
}

func (id *Id) Set(value string) (err error) {
	if len(value) == 0 {
		err = errors.Errorf("empty blob_store_id")
		return err
	}

	firstChar := rune(value[0])

	if id.location.IsPrefix(firstChar) {
		id.id = value[1:]

		if err = id.location.SetPrefix(firstChar); err != nil {
			err = errors.Errorf(
				"unsupported first char for blob_store_id: %q",
				string(firstChar),
			)

			return err
		}
	} else {
		id.location = LocationTypeXDGUser
		id.id = value
	}

	return err
}

func (id Id) Less(otherId Id) bool {
	if id.location < otherId.location {
		return true
	}

	return id.id < otherId.id
}

func (id Id) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *Id) UnmarshalText(bites []byte) (err error) {
	if err = id.Set(string(bites)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

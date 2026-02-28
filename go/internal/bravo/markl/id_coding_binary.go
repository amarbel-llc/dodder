package markl

import (
	"bytes"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type IdBinaryDecodingTypeData struct {
	domain_interfaces.MarklIdMutable
}

func (id IdBinaryDecodingTypeData) UnmarshalBinary(
	bites []byte,
) (err error) {
	if len(bites) == 0 {
		return err
	}

	formatIdBytes, bytesAfterFormatId, ok := bytes.Cut(bites, []byte{'\x00'})

	if !ok {
		err = errors.Errorf("expected empty byte, but none found")
		return err
	}

	if err = id.SetMarklId(string(formatIdBytes), bytesAfterFormatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

type IdBinaryDecodingFormatTypeData struct {
	domain_interfaces.MarklIdMutable
}

func (id *Id) UnmarshalBinary(
	bites []byte,
) (err error) {
	if len(bites) == 0 {
		return err
	}

	formatBytes, bytesAfterFormat, ok := bytes.Cut(bites, []byte{'\x00'})

	if !ok {
		err = errors.Errorf("expected empty byte, but none found")
		return err
	}

	if err = id.SetPurposeId(string(formatBytes)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	formatIdBytes, bytesAfterFormatId, ok := bytes.Cut(
		bytesAfterFormat,
		[]byte{'\x00'},
	)

	if !ok {
		err = errors.Errorf("expected empty byte, but none found")
		return err
	}

	if err = id.SetMarklId(string(formatIdBytes), bytesAfterFormatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

type IdBinaryEncodingTypeData struct {
	domain_interfaces.MarklId
}

// structure (in bytes):
// <256: type
// 1: empty byte
// <256: id
func (id IdBinaryEncodingTypeData) MarshalBinary() (bytes []byte, err error) {
	// TODO confirm few allocations
	// TODO confirm size of type is less than 256
	format := id.GetMarklFormat()
	bites := id.GetBytes()

	if format == nil && len(bites) == 0 {
		return bytes, err
	} else if format == nil {
		err = errors.Errorf("empty format for markl id")
		return bytes, err
	}

	bytes = append(bytes, []byte(format.GetMarklFormatId())...)
	bytes = append(bytes, '\x00')
	bytes = append(bytes, bites...)

	return bytes, err
}

type IdBinaryEncodingFormatTypeData struct {
	domain_interfaces.MarklId
}

// structure (in bytes):
// <256: type
// 1: empty byte
// <256: id
func (id Id) MarshalBinary() (bytes []byte, err error) {
	// TODO confirm few allocations
	// TODO confirm size of type is less than 256
	purpose := id.GetPurposeId()
	format := id.GetMarklFormat()
	bites := id.GetBytes()

	if format == nil && len(bites) == 0 && purpose == "" {
		return bytes, err
	} else if format == nil {
		err = errors.Errorf("empty type")
		return bytes, err
	}

	bytes = append(bytes, []byte(purpose)...)
	bytes = append(bytes, '\x00')
	bytes = append(bytes, []byte(format.GetMarklFormatId())...)
	bytes = append(bytes, '\x00')
	bytes = append(bytes, bites...)

	return bytes, err
}

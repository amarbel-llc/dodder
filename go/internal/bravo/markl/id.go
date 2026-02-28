package markl

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/blech32"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

var (
	_ domain_interfaces.MarklId        = Id{}
	_ domain_interfaces.MarklIdMutable = &Id{}
)

type Id struct {
	purposeId string
	format    domain_interfaces.MarklFormat
	data      []byte
}

func (id Id) String() string {
	emptyFormat := id.format == nil
	emptyData := len(id.data) == 0
	var hrp string

	switch {
	case emptyFormat && emptyData:
		return ""

	case emptyData:
		return ""

	case emptyFormat:
		hrp = "!error-empty_format"
		fallthrough

	default:
		if hrp == "" {
			hrp = id.format.GetMarklFormatId()
		}

		bites, err := blech32.Encode(hrp, id.data)
		errors.PanicIfError(err)
		return string(bites)
	}
}

func (id Id) StringWithFormat() string {
	if id.format == nil && len(id.data) == 0 {
		return ""
	}

	if len(id.data) == 0 {
		return ""
	} else {
		bites, err := blech32.Encode(id.format.GetMarklFormatId(), id.data)
		bitesString := string(bites)
		errors.PanicIfError(err)

		if id.purposeId != "" {
			return fmt.Sprintf("%s@%s", id.purposeId, bitesString)
		} else {
			return bitesString
		}
	}
}

func (id Id) GetPurposeId() string {
	return id.purposeId
}

func (id *Id) SetPurposeId(value string) error {
	id.purposeId = value
	return nil
}

func (id Id) IsEmpty() bool {
	return len(id.data) == 0
}

func (id Id) GetSize() int {
	return len(id.data)
}

func (id Id) GetBytes() []byte {
	return id.data
}

func (id Id) GetMarklFormat() domain_interfaces.MarklFormat {
	return id.format
}

func (id Id) IsNull() bool {
	if len(id.data) == 0 {
		return true
	} else if id.format == nil {
		panic("empty type")
	}

	formatHash, ok := formatHashes[id.format.GetMarklFormatId()]

	// this is not an Id for a hash, so it can never be null with non-zero data
	// contents
	if !ok {
		return false
	}

	if bytes.Equal(id.data, formatHash.null.GetBytes()) {
		return true
	}

	return false
}

func (id *Id) Set(value string) (err error) {
	purpose, body, ok := strings.Cut(value, "@")

	if ok {
		if err = id.setWithPurpose(purpose, body); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = id.setWithoutPurpose(value); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (id *Id) setWithPurpose(purpose, body string) (err error) {
	if err = id.SetPurposeId(purpose); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.setWithoutPurpose(body); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) setWithoutPurpose(value string) (err error) {
	var formatId string

	if formatId, id.data, err = blech32.DecodeString(value); err != nil {
		err = errors.Wrapf(err, "Value: %q", value)
		return err
	}

	if err = id.SetMarklId(formatId, id.data); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) SetDigest(digest domain_interfaces.MarklId) (err error) {
	if err = id.SetPurposeId(digest.GetPurposeId()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.SetMarklId(
		digest.GetMarklFormat().GetMarklFormatId(),
		digest.GetBytes(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) setFormatId(formatId string) (err error) {
	if id.format, err = GetFormatOrError(formatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = validatePurposeAndFormatId(id.purposeId, formatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) SetMarklId(formatId string, bites []byte) (err error) {
	if formatId == "" && len(bites) == 0 {
		id.Reset()
		return err
	}

	if err = id.setFormatId(formatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.setData(bites); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) allocDataIfNecessary(size int) {
	id.data = id.data[:0]
	id.data = slices.Grow(id.data, size)
}

func (id *Id) allocDataAndSetToCapIfNecessary(size int) {
	id.allocDataIfNecessary(size)
	id.data = id.data[:size]
}

func (id *Id) setData(bites []byte) (err error) {
	// empty is permitted
	if len(bites) == 0 {
		return err
	}

	// TODO enforce non-nil formats
	if id.format != nil {
		expected := id.format.GetSize()
		actual := len(bites)

		if actual != expected {
			err = errors.Errorf(
				"wrong size for bytes: expected %d, but got %d",
				expected,
				actual,
			)

			return err
		}
	}

	id.allocDataAndSetToCapIfNecessary(len(bites))
	copy(id.data, bites)

	return err
}

func (id *Id) Reset() {
	id.format = nil
	id.data = id.data[:0]
	id.purposeId = ""
}

func (id *Id) ResetWithPurpose(purpose string) {
	id.format = nil
	id.data = id.data[:0]
	id.purposeId = purpose
}

func (id *Id) ResetWith(src Id) {
	id.purposeId = src.purposeId
	id.format = src.format
	errors.PanicIfError(id.setData(src.data))
}

func (id *Id) ResetWithOrDefaultPurpose(src Id, purpose string) {
	if src.IsEmpty() {
		id.ResetWithPurpose(purpose)
	} else {
		id.ResetWith(src)
	}
}

func (id *Id) ResetWithMarklId(src domain_interfaces.MarklId) {
	marklType := src.GetMarklFormat()
	bites := src.GetBytes()

	var marklTypeId string

	if marklType == nil && len(bites) > 0 {
		panic(
			fmt.Sprintf(
				"markl id with empty format but populated bytes: (bites: %x)",
				bites,
			),
		)
	} else if marklType != nil {
		marklTypeId = marklType.GetMarklFormatId()
	}

	errors.PanicIfError(
		id.SetPurposeId(src.GetPurposeId()),
	)

	errors.PanicIfError(
		id.SetMarklId(marklTypeId, bites),
	)
}

func (id *Id) GetBlobId() domain_interfaces.MarklId {
	return id
}

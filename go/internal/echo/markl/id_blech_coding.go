package markl

import (
	"encoding/hex"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/blech32"
)

// TODO remove
func SetMaybeSha256(id domain_interfaces.MarklIdMutable, value string) (err error) {
	// TODO use registered format lengths
	switch len(value) {
	case 65:
		if value[0] != '@' {
			err = errors.Errorf("unknown format: %q", value)
			return err
		}

		value = strings.TrimPrefix(value, "@")
		fallthrough

	case 64:
		if err = setSha256(id, value); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		if err = id.Set(value); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func SetMarklIdWithFormatBlech32(
	id domain_interfaces.MarklIdMutable,
	purposeId string,
	blechValue string,
) (err error) {
	if err = id.SetPurposeId(purposeId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.Set(
		blechValue,
	); errors.Is(err, blech32.ErrSeparatorMissing) {
		if err = setSha256(
			id,
			blechValue,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else if err != nil {
		err = errors.Wrap(err)
		return err
	}

	formatId := id.GetMarklFormat().GetMarklFormatId()

	if err = validatePurposeAndFormatId(purposeId, formatId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func validatePurposeAndFormatId(purposeId string, formatId string) (err error) {
	if formatId == "" || purposeId == "" {
		return err
	}

	purpose := GetPurpose(purposeId)

	if _, ok := purpose.formatIds[formatId]; !ok {
		err = errors.Errorf("format id %q not supported for purpose %q", formatId, purposeId)
		return err
	}

	return err
}

// TODO remove
func setSha256(id domain_interfaces.MarklIdMutable, value string) (err error) {
	var decodedBytes []byte

	if decodedBytes, err = hex.DecodeString(value); err != nil {
		err = errors.Wrapf(err, "%q", value)
		return err
	}

	if err = id.SetMarklId(
		FormatIdHashSha256,
		decodedBytes,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

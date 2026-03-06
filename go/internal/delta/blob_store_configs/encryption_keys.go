package blob_store_configs

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/age"
)

type EncryptionKeys []markl.Id

var _ domain_interfaces.MarklId = EncryptionKeys{}

func (keys EncryptionKeys) String() string {
	return keys.StringWithFormat()
}

func (keys EncryptionKeys) StringWithFormat() string {
	parts := make([]string, 0, len(keys))

	for _, key := range keys {
		s := key.StringWithFormat()

		if s != "" {
			parts = append(parts, s)
		}
	}

	return strings.Join(parts, ", ")
}

func (keys EncryptionKeys) GetBytes() []byte {
	if len(keys) == 0 {
		return nil
	}

	return keys[0].GetBytes()
}

func (keys EncryptionKeys) GetSize() int {
	if len(keys) == 0 {
		return 0
	}

	return keys[0].GetSize()
}

func (keys EncryptionKeys) MarshalBinary() ([]byte, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	return keys[0].MarshalBinary()
}

func (keys EncryptionKeys) GetMarklFormat() domain_interfaces.MarklFormat {
	if len(keys) == 0 {
		return nil
	}

	return keys[0].GetMarklFormat()
}

func (keys EncryptionKeys) IsNull() bool {
	if len(keys) == 0 {
		return true
	}

	for _, key := range keys {
		if !key.IsNull() {
			return false
		}
	}

	return true
}

func (keys EncryptionKeys) IsEmpty() bool {
	return len(keys) == 0
}

func (keys EncryptionKeys) GetPurposeId() string {
	if len(keys) == 0 {
		return ""
	}

	return keys[0].GetPurposeId()
}

func (keys EncryptionKeys) GetIOWrapper() (
	ioWrapper interfaces.IOWrapper,
	err error,
) {
	nonNullKeys := make([]markl.Id, 0, len(keys))

	for _, key := range keys {
		if !key.IsNull() {
			nonNullKeys = append(nonNullKeys, key)
		}
	}

	if len(nonNullKeys) == 0 {
		return nil, nil
	}

	identities := make([]age.Identity, 0, len(nonNullKeys))

	for _, key := range nonNullKeys {
		var keyWrapper interfaces.IOWrapper

		if keyWrapper, err = key.GetIOWrapper(); err != nil {
			err = errors.Wrap(err)
			return ioWrapper, err
		}

		ageIdentity, ok := keyWrapper.(*age.Identity)

		if !ok {
			err = errors.Errorf("expected *age.Identity, got %T", keyWrapper)
			return ioWrapper, err
		}

		identities = append(identities, *ageIdentity)
	}

	mi := age.MakeMultiIdentity(identities)
	ioWrapper = &mi

	return ioWrapper, err
}

func (keys EncryptionKeys) Verify(
	mes, sig domain_interfaces.MarklId,
) error {
	return errors.Errorf("Verify not supported on EncryptionKeys")
}

func (keys EncryptionKeys) Sign(
	mes domain_interfaces.MarklId,
	sigDst domain_interfaces.MarklIdMutable,
	sigPurpose string,
) error {
	return errors.Errorf("Sign not supported on EncryptionKeys")
}

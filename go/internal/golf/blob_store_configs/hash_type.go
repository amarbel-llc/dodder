package blob_store_configs

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type ErrUnsupportedHashType string

func (err ErrUnsupportedHashType) Error() string {
	return fmt.Sprintf("unsupported hash type: %q", string(err))
}

func (ErrUnsupportedHashType) Is(target error) (ok bool) {
	_, ok = target.(ErrUnsupportedHashType)
	return ok
}

type HashType string

const (
	HashTypeSha256     = HashType(markl.FormatIdHashSha256)
	HashTypeBlake2b256 = HashType(markl.FormatIdHashBlake2b256)

	HashTypeDefault = HashTypeBlake2b256
)

func (hashType HashType) String() string {
	return string(hashType)
}

func (hashType *HashType) Set(value string) error {
	valueClean := HashType(strings.TrimSpace(strings.ToLower(value)))

	switch valueClean {
	case HashTypeSha256, HashTypeBlake2b256:
		*hashType = valueClean

	default:
		return ErrUnsupportedHashType(value)
	}

	return nil
}

func (hashType HashType) GetCLICompletion() map[string]string {
	return map[string]string{
		HashTypeBlake2b256.String(): "BLAKE2b-256 (default)",
		HashTypeSha256.String():     "SHA-256",
	}
}

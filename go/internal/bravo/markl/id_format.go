package markl

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
)

type ErrUnsupportedIdFormat string

func (err ErrUnsupportedIdFormat) Error() string {
	return fmt.Sprintf("unsupported id format: %q", string(err))
}

func (ErrUnsupportedIdFormat) Is(target error) (ok bool) {
	_, ok = target.(ErrUnsupportedIdFormat)
	return ok
}

type IdFormat string

const (
	IdFormatBlech32 = IdFormat("blech32")
	IdFormatHex     = IdFormat("hex")

	IdFormatDefault = IdFormatBlech32
)

func (idFormat IdFormat) String() string {
	return string(idFormat)
}

func (idFormat *IdFormat) Set(value string) error {
	valueClean := IdFormat(strings.TrimSpace(strings.ToLower(value)))

	switch valueClean {
	case IdFormatBlech32, IdFormatHex:
		*idFormat = valueClean

	default:
		return ErrUnsupportedIdFormat(value)
	}

	return nil
}

func (idFormat IdFormat) GetCLICompletion() map[string]string {
	return map[string]string{
		IdFormatBlech32.String(): "blech32 encoding (default)",
		IdFormatHex.String():     "hexadecimal encoding",
	}
}

func (idFormat IdFormat) FormatId(id domain_interfaces.MarklId) string {
	switch idFormat {
	case IdFormatHex:
		return fmt.Sprintf(
			"%s-%s",
			id.GetMarklFormat().GetMarklFormatId(),
			FormatBytesAsHex(id),
		)

	default:
		return id.String()
	}
}

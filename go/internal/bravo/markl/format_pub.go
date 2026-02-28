package markl

import "code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"

type (
	FuncFormatPubVerify func(pubkey, message, sig domain_interfaces.MarklId) error

	FormatPub struct {
		Id   string
		Size int

		Verify FuncFormatPubVerify
	}
)

var _ domain_interfaces.MarklFormat = FormatPub{}

func (format FormatPub) GetMarklFormatId() string {
	return format.Id
}

func (format FormatPub) GetSize() int {
	return format.Size
}

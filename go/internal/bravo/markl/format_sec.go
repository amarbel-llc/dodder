package markl

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type (
	// TODO switch to accepting bytes?
	FuncFormatSecGenerate     func(io.Reader) ([]byte, error)
	FuncFormatSecGetPublicKey func(private domain_interfaces.MarklId) ([]byte, error)
	FuncFormatSecGetIOWrapper func(private domain_interfaces.MarklId) (interfaces.IOWrapper, error)
	FuncFormatSecSign         func(sec, mes domain_interfaces.MarklId, readerRand io.Reader) ([]byte, error)

	FormatSec struct {
		Id   string
		Size int

		Generate FuncFormatSecGenerate

		PubFormatId  string
		GetPublicKey FuncFormatSecGetPublicKey

		GetIOWrapper FuncFormatSecGetIOWrapper

		SigFormatId string
		Sign        FuncFormatSecSign
	}
)

var _ domain_interfaces.MarklFormat = FormatSec{}

func (format FormatSec) GetMarklFormatId() string {
	return format.Id
}

func (format FormatSec) GetSize() int {
	return format.Size
}

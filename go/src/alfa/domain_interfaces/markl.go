package domain_interfaces

import (
	"encoding"
	"hash"
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

type (
	MarklFormat interface {
		GetSize() int
		GetMarklFormatId() string
	}

	FormatHash interface {
		MarklFormat

		GetHash() (Hash, interfaces.FuncRepool)

		GetMarklIdForString(input string) (MarklId, interfaces.FuncRepool)
		GetMarklIdFromStringFormat(
			format string,
			args ...any,
		) (MarklId, interfaces.FuncRepool)
	}

	MarklFormatGetter interface {
		GetMarklFormat() MarklFormat
	}

	Hash interface {
		hash.Hash
		MarklFormatGetter
		// TODO add `WriteToMarklId` method for reuse
		GetMarklId() (MarklIdMutable, interfaces.FuncRepool)
	}

	MarklId interface {
		// TODO consider removing Stringer and Setter

		// TODO add WriteString and WriteStringWithFormat
		interfaces.Stringer
		StringWithFormat() string

		encoding.BinaryMarshaler
		// encoding.TextMarshaler
		// io.WriterTo
		GetBytes() []byte
		// TODO rethink size as it works completely different between sha and
		// merkle
		GetSize() int
		MarklFormatGetter
		IsNull() bool
		IsEmpty() bool

		GetPurposeId() string

		// Optional methods
		GetIOWrapper() (interfaces.IOWrapper, error)
		Verify(mes, sig MarklId) error
		Sign(
			mes MarklId,
			sigDst MarklIdMutable,
			sigPurpose string,
		) (err error)
	}

	MarklIdMutable interface {
		MarklId
		interfaces.Setter
		encoding.BinaryUnmarshaler
		// encoding.TextUnmarshaler
		// io.ReaderFrom
		SetMarklId(formatId string, bites []byte) error
		Reset()
		ResetWithMarklId(MarklId)
		SetPurposeId(string) error

		// Optional methods
		GeneratePrivateKey(
			readerRand io.Reader,
			formatId string,
			purpose string,
		) (err error)
	}

	MarklIdGetter interface {
		GetMarklId() MarklId
	}

	DigestWriteMap map[string]MarklIdMutable

	Lock[
		KEY interfaces.Value,
		KEY_PTR interfaces.ValuePtr[KEY],
	] interface {
		GetKey() KEY
		GetValue() MarklId
		IsEmpty() bool
	}

	LockMutable[
		KEY interfaces.Value,
		KEY_PTR interfaces.ValuePtr[KEY],
	] interface {
		Lock[KEY, KEY_PTR]
		GetKeyMutable() KEY_PTR
		GetValueMutable() MarklIdMutable
	}
)

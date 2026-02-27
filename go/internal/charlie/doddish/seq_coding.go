package doddish

import (
	"slices"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
)

type SeqBinaryCoding Seq

func (marshaler SeqBinaryCoding) ToSeq() Seq {
	return Seq(marshaler)
}

func (marshaler *SeqBinaryCoding) ToSeqMutable() *Seq {
	return (*Seq)(marshaler)
}

func (marshaler SeqBinaryCoding) MarshalBinary() (bites []byte, err error) {
	return marshaler.AppendBinary(nil)
}

func (marshaler SeqBinaryCoding) AppendBinary(bites []byte) ([]byte, error) {
	var byteCount int

	for _, token := range marshaler.ToSeq() {
		byteCount += token.GetBinaryByteCount()
	}

	bites = slices.Grow(bites, byteCount)

	for _, token := range marshaler.ToSeq() {
		added := ohio.UInt8ToByteArray(uint8(token.GetBinaryByteCount()))[0]
		bites = append(bites, added)

		var err error

		if bites, err = token.AppendBinary(bites); err != nil {
			err = errors.Wrap(err)
			return bites, err
		}
	}

	return bites, nil
}

func (marshaler *SeqBinaryCoding) UnmarshalBinary(bites []byte) (err error) {
	for len(bites) > 0 {
		byteCount := ohio.ByteArrayToUInt8([1]byte{bites[0]})

		bites = bites[1:]

		if len(bites) == 0 {
			return
		}

		if int(byteCount) > len(bites) {
			err = errors.Errorf(
				"expected at least %d bytes, but have %d bytes",
				byteCount,
				len(bites),
			)

			return
		}

		tokenBytes := bites[:byteCount]
		bites = bites[byteCount:]

		// TODO use token byte pool?
		var token Token

		if err = token.UnmarshalBinary(tokenBytes); err != nil {
			err = errors.Wrap(err)
			return err
		}

		marshaler.ToSeqMutable().GetSliceMutable().Append(token)
	}

	return err
}

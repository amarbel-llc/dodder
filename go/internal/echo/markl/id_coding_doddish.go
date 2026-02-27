package markl

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
)

type idCoderDoddish Id

func MakeIdCoderDoddish(id *Id) *idCoderDoddish {
	return (*idCoderDoddish)(id)
}

// var _ interfaces.Coder[Id, *Id] = idCoderDoddish{}
func (coder *idCoderDoddish) GetIdMutable() *Id {
	return (*Id)(coder)
}

func (coder idCoderDoddish) MarshalDoddish() (doddish.Seq, error) {
	return nil, errors.Err501NotImplemented
}

func (coder idCoderDoddish) UnmarshalDoddish(seq doddish.Seq) (err error) {
	if !seq.MatchAll(doddish.TokenMatcherDodderTag...) {
		err = newPkgError("unsupported seq")
		return err
	}

	var purposeId string
	var value []byte

	if seq.Len() == 3 {
		purposeId = string(seq.At(0).Contents)
		value = seq.At(2).Contents
	} else {
		// blobs have only the `@` symbol and no purpose, so we implicitly decide
		// the purpose when parsing
		purposeId = PurposeBlobDigestV1
		value = seq.At(1).Contents
	}

	if err = SetMarklIdWithFormatBlech32(
		coder.GetIdMutable(),
		purposeId,
		string(value),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

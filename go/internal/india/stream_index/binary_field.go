package stream_index

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"code.linenisgreat.com/dodder/go/internal/_/key_bytes"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

var binaryFieldOrder = []key_bytes.Binary{
	key_bytes.Sigil,
	key_bytes.ObjectId,
	key_bytes.Blob,
	key_bytes.RepoPubKey,
	key_bytes.RepoSig,
	key_bytes.Description,
	key_bytes.Tag,
	key_bytes.Tai,
	key_bytes.Type,
	key_bytes.SigParentMetadataParentObjectId,
	key_bytes.DigestMetadataParentObjectId,
	key_bytes.DigestMetadataWithoutTai,
	key_bytes.CacheTagImplicit,
	key_bytes.CacheTags,
}

type binaryField struct {
	Key           key_bytes.Binary
	ContentLength uint16
	Content       bytes.Buffer
}

func (binaryField *binaryField) String() string {
	return fmt.Sprintf(
		"%s:%d:%x",
		binaryField.Key,
		binaryField.ContentLength,
		binaryField.Content.Bytes(),
	)
}

func (binaryField *binaryField) Reset() {
	binaryField.Key.Reset()
	binaryField.ContentLength = 0
	binaryField.Content.Reset()
}

var errContentLengthTooLarge = newPkgError("content length too large")

func (binaryField *binaryField) ReadFrom(r io.Reader) (n int64, err error) {
	var n1 int
	var n2 int64
	n2, err = binaryField.Key.ReadFrom(r)
	n += int64(n2)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	n1, binaryField.ContentLength, err = ohio.ReadFixedUInt16(r)
	n += int64(n1)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	binaryField.Content.Grow(int(binaryField.ContentLength))
	binaryField.Content.Reset()

	n2, err = io.CopyN(
		&binaryField.Content,
		r,
		int64(binaryField.ContentLength),
	)
	n += n2

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (binaryField *binaryField) WriteTo(w io.Writer) (n int64, err error) {
	// defer func() {
	// 	r := recover()

	// 	if r == nil {
	// 		return
	// 	}

	// 	ui.Debug().Print(bf.Content.Len(), bf.ContentLength)
	// 	panic(r)
	// }()

	if binaryField.Content.Len() > math.MaxUint16 {
		err = errContentLengthTooLarge
		return n, err
	}

	binaryField.ContentLength = uint16(binaryField.Content.Len())

	var n1 int
	var n2 int64
	n2, err = binaryField.Key.WriteTo(w)
	n += int64(n2)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	n1, err = ohio.WriteFixedUInt16(w, binaryField.ContentLength)
	n += int64(n1)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	n2, err = io.Copy(w, &binaryField.Content)
	n += n2

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}

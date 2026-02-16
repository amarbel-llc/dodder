package ids

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/cmp"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/alfa/quiter_collection"
	"code.linenisgreat.com/dodder/go/src/bravo/comments"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/charlie/doddish"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
)

var poolObjectId3 = pool.Make(
	nil,
	func(id *objectId3) {
		id.Reset()
	},
)

func getObjectIdPool3() interfaces.PoolPtr[objectId3, *objectId3] {
	return poolObjectId3
}

// TODO add support for binary marshaling
// TODO make fields private
type objectId3 struct {
	Genre genres.Genre
	Seq   doddish.Seq
}

var _ Id = objectId3{}

func (id *objectId3) GetObjectId() *objectId3 {
	return id
}

func (id objectId3) GetGenre() domain_interfaces.Genre {
	return id.Genre
}

func (id objectId3) IsEmpty() bool {
	for _, token := range id.Seq {
		if token.Type == doddish.TokenTypeIdentifier && len(token.Contents) > 0 {
			return false
		}
	}

	return true
}

func (id objectId3) Len() int {
	var count int

	for _, token := range id.Seq {
		count += len(token.Contents)
	}

	return count
}

func (id objectId3) String() string {
	return id.Seq.String()
}

func (id objectId3) StringSansRepo() string {
	return id.String()
}

func (id objectId3) StringSansOp() string {
	seq := id.Seq

	if seq.Len() > 0 && seq.At(0).Type == doddish.TokenTypeOperator {
		seq = seq[1:]
	}

	return seq.String()
}

func (id objectId3) Equals(other objectId3) bool {
	if id.Genre != other.Genre {
		return false
	}

	if !quiter_collection.Equals(id.Seq.GetSlice(), other.Seq.GetSlice()) {
		return false
	}

	return true
}

func (id *objectId3) SetWithId(otherId Id) (err error) {
	return id.SetWithSeq(otherId.ToSeq())
}

func (id *objectId3) Set(value string) (err error) {
	var seq doddish.Seq

	if seq, err = doddish.ScanExactlyOneSeqWithDotAllowedInIdenfierFromString(
		value,
	); err != nil {
		if id.Genre == genres.Unknown {
			return id.SetBlob(value)
		}

		err = errors.Wrap(err)
		return err
	}

	return id.SetWithSeq(seq)
}

func (id *objectId3) SetWithSeq(seq doddish.Seq) (err error) {
	if id.Genre, err = ValidateSeqAndGetGenre(seq); err != nil {
		err = errors.Wrap(err)
		return
	}

	id.Seq = seq.Clone()

	return
}

func (id *objectId3) Reset() {
	id.Genre = genres.Unknown
	id.Seq.GetSliceMutable().Reset()
}

func (id *objectId3) ResetWith(other objectId3) {
	id.Genre = other.Genre

	comments.Performance("switch to reusing exising seq's data")
	id.Seq = other.Seq.Clone()
}

func (id *objectId3) ResetWithObjectId(other Id) {
	switch other := other.(type) {
	case TypeStruct:
		id.ResetWithType(other)

	case *TypeStruct:
		id.ResetWithType(*other)

	default:
		id.Genre = genres.Must(other.GetGenre())
		id.Seq = other.ToSeq().Clone()
	}
}

func (id *objectId3) ResetWithType(other TypeStruct) {
	id.Genre = genres.Type
	id.Seq = doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeOperator,
			Contents: []byte("!"),
		},
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(other.StringSansOp()),
		},
	}
}

func (id objectId3) ToType() TypeStruct {
	if id.IsEmpty() {
		return TypeStruct{}
	} else if !id.Genre.IsType() {
		panic(fmt.Sprintf("id is not a type, genre is %q: %q", id.Genre, id))
	}

	return TypeStruct{Value: id.Seq.At(1).String()}
}

func SeqIdCompare(left, right objectId3) cmp.Result {
	return doddish.SeqCompare(left.Seq, right.Seq)
}

func (id *objectId3) SetType(value string) (err error) {
	reader, repool := pool.GetStringReader(value)
	defer repool()

	boxScanner := doddish.MakeScanner(reader)

	var seq doddish.Seq

	if seq, err = boxScanner.ScanDotAllowedInIdentifiersOrError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	switch {
	default:
		err = errors.Wrap(doddish.ErrUnsupportedSeq{Seq: seq, For: "objectId3.SetType"})
		return err

	case seq.Len() == 0:
		err = doddish.ErrEmptySeq
		return err

		// type
	case seq.MatchAll(doddish.TokenTypeIdentifier):
		if TokenIsConfig(seq.At(0)) {
			err = errors.Errorf("cannot use konfig identifier: %q", seq)
			return err
		}

		id.Genre = genres.Type
		seq = doddish.Seq{
			doddish.Token{
				Type:     doddish.TokenTypeOperator,
				Contents: []byte("!"),
			},
			seq.At(0),
		}

		// .type
	case seq.MatchAll(
		doddish.TokenMatcherOp(doddish.OpSigilExternal),
		doddish.TokenTypeIdentifier,
	):
		if TokenIsConfig(seq.At(1)) {
			err = errors.Errorf("cannot use konfig identifier: %q", seq)
			return err
		}

		id.Genre = genres.Type
		seq = doddish.Seq{
			doddish.Token{
				Type:     doddish.TokenTypeOperator,
				Contents: []byte("!"),
			},
			seq.At(0),
		}

		// !type
	case seq.MatchAll(
		doddish.TokenMatcherOp(doddish.OpType),
		doddish.TokenTypeIdentifier,
	):
		if TokenIsConfig(seq.At(1)) {
			err = errors.Errorf("cannot use konfig identifier: %q", seq)
			return err
		}

		id.Genre = genres.Type
	}

	id.Seq = seq.Clone()

	return err
}

func (id objectId3) ToSeq() doddish.Seq {
	return id.Seq
}

func (id *objectId3) ToSeqMutable() *doddish.Seq {
	return &id.Seq
}

func (id objectId3) MarshalBinary() ([]byte, error) {
	return id.AppendBinary(nil)
}

func (id objectId3) AppendBinary(bites []byte) ([]byte, error) {
	return id.ToSeq().GetBinaryMarshaler().AppendBinary(bites)
}

func (id *objectId3) UnmarshalBinary(bites []byte) (err error) {
	if err = id.ToSeqMutable().GetBinaryUnmarshaler().UnmarshalBinary(
		bites,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if id.Genre, err = ValidateSeqAndGetGenre(id.Seq); err != nil {
		if doddish.IsErrEmptySeq(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	return nil
}

func (id *objectId3) Clone() (clone *objectId3, repool interfaces.FuncRepool) {
	clone, repool = getObjectIdPool3().GetWithRepool()
	clone.ResetWithObjectId(id)
	return clone, repool
}

func (id objectId3) MarshalText() (text []byte, err error) {
	text = []byte(FormattedString(id))
	return text, err
}

func (id *objectId3) UnmarshalText(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *objectId3) WriteTo(w io.Writer) (n int64, err error) {
	var n1 int64
	n1, err = id.Genre.WriteTo(w)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var seqBytes []byte

	if seqBytes, err = id.Seq.GetBinaryMarshaler().MarshalBinary(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	b := [1]uint8{uint8(len(seqBytes))}

	var n2 int
	n2, err = ohio.WriteAllOrDieTrying(w, b[:])
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = ohio.WriteAllOrDieTrying(w, seqBytes)
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (id *objectId3) ReadFrom(r io.Reader) (n int64, err error) {
	id.Seq.GetSliceMutable().Reset()

	var n1 int64
	n1, err = id.Genre.ReadFrom(r)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var b [1]uint8

	var n2 int
	n2, err = ohio.ReadAllOrDieTrying(r, b[:])
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	seqLen := b[0]

	if seqLen == 0 {
		return n, err
	}

	seqBytes := make([]byte, seqLen)

	n2, err = ohio.ReadAllOrDieTrying(r, seqBytes)
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = id.Seq.GetBinaryUnmarshaler().UnmarshalBinary(seqBytes); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (id *objectId3) SetGenre(genre domain_interfaces.GenreGetter) {
	if genre == nil {
		id.Genre = genres.Unknown
	} else {
		id.Genre = genres.Must(genre.GetGenre())
	}
}

func (id *objectId3) SetBlob(value string) (err error) {
	id.Genre = genres.Blob

	var seq doddish.Seq

	if seq, err = doddish.ScanExactlyOneSeqWithDotAllowedInIdenfierFromString(
		value,
	); err != nil {
		// Fall back to creating a single identifier token from the raw value
		// for multi-word blob names like "to add"
		id.Seq = doddish.Seq{
			doddish.Token{
				Type:     doddish.TokenTypeIdentifier,
				Contents: []byte(value),
			},
		}

		return nil
	}

	id.Seq = seq.Clone()

	return err
}

func (id *objectId3) SetWithGenre(
	value string,
	genre domain_interfaces.GenreGetter,
) (err error) {
	g := genres.Make(genre.GetGenre())

	switch g {
	case genres.Type:
		return id.SetType(value)

	case genres.Blob:
		return id.SetBlob(value)

	default:
		if err = id.Set(value); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}

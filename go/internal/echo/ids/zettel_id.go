package ids

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/coordinates"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

func init() {
	register(ZettelId{})
}

type ZettelId struct {
	left, right string
}

type Provider interface {
	MakeZettelIdFromCoordinates(i coordinates.Int) (string, error)
}

func NewZettelIdEmpty() ZettelId {
	return ZettelId{}
}

// TODO-P3 is this really necessary?;w
func MakeZettelIdFromProvidersAndCoordinates(
	coordinate coordinates.Int,
	leftProvider Provider,
	rightProvider Provider,
) (h *ZettelId, err error) {
	id := coordinates.ZettelIdCoordinate{}
	id.SetInt(coordinate)

	var left, right string

	if left, err = leftProvider.MakeZettelIdFromCoordinates(id.Left); err != nil {
		err = errors.ErrorWithStackf("failed to make left zettel id: %s", err)
		return h, err
	}

	if right, err = rightProvider.MakeZettelIdFromCoordinates(id.Right); err != nil {
		err = errors.ErrorWithStackf("failed to make right zettel id: %s", err)
		return h, err
	}

	return MakeZettelIdFromHeadAndTail(left, right)
}

func MakeZettelIdFromHeadAndTail(head, tail string) (h *ZettelId, err error) {
	head = strings.TrimSpace(head)
	tail = strings.TrimSpace(tail)

	switch {
	case head == "":
		err = errors.ErrorWithStackf(
			"head was empty: {head: %q, tail: %q}",
			head,
			tail,
		)
		return h, err

	case tail == "":
		err = errors.ErrorWithStackf(
			"tail was empty: {head: %q, tail: %q}",
			head,
			tail,
		)
		return h, err
	}

	hs := fmt.Sprintf("%s/%s", head, tail)

	h = &ZettelId{}

	if err = h.Set(hs); err != nil {
		err = errors.ErrorWithStackf("failed to set zettel id: %s", err)
		return h, err
	}

	return h, err
}

func MustZettelId(v string) (h *ZettelId) {
	var err error
	h, err = MakeZettelId(v)

	errors.PanicIfError(err)

	return h
}

func MakeZettelId(v string) (h *ZettelId, err error) {
	h = &ZettelId{}

	if err = h.Set(v); err != nil {
		return h, err
	}

	return h, err
}

func (id ZettelId) IsEmpty() bool {
	return id.left == "" && id.right == ""
}

func (id ZettelId) Equals(b ZettelId) bool {
	if id.left != b.left {
		return false
	}

	if id.right != b.right {
		return false
	}

	return true
}

func (id ZettelId) GetHead() string {
	return id.left
}

func (id ZettelId) GetTail() string {
	return id.right
}

func (id ZettelId) GetObjectIdString() string {
	return id.String()
}

func (id ZettelId) String() string {
	v := fmt.Sprintf("%s/%s", id.left, id.right)
	return v
}

func (id ZettelId) Less(j ZettelId) bool {
	return id.String() < j.String()
}

func (id *ZettelId) SetWithSeq(seq doddish.Seq) (err error) {
	switch {
	case seq.MatchAll(
		doddish.TokenMatcherOp(doddish.OpPathSeparator),
	):

	case seq.MatchAll(
		doddish.TokenTypeIdentifier,
		doddish.TokenMatcherOp(doddish.OpPathSeparator),
		doddish.TokenTypeIdentifier,
	):
		id.left = seq.At(0).String()
		id.right = seq.At(2).String()

	default:
		err = errors.Errorf("seq isn't a zettel id: %q", seq)
		return err
	}

	return err
}

// TODO switch to doddish.Seq
func (id *ZettelId) Set(value string) (err error) {
	var seq doddish.Seq

	if seq, err = doddish.ScanExactlyOneSeqWithDotAllowedInIdenfierFromString(
		value,
	); err != nil {
		if doddish.IsErrEmptySeq(err) {
			err = nil
			id.Reset()
			return
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = id.SetWithSeq(seq); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *ZettelId) Reset() {
	id.left = ""
	id.right = ""
}

func (id *ZettelId) ResetWith(h1 ZettelId) {
	id.left = h1.left
	id.right = h1.right
}

func (id ZettelId) GetGenre() domain_interfaces.Genre {
	return genres.Zettel
}

func (id ZettelId) MarshalText() (text []byte, err error) {
	text = []byte(id.String())
	return text, err
}

func (id *ZettelId) UnmarshalText(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id ZettelId) MarshalBinary() (text []byte, err error) {
	return id.AppendBinary(nil)
}

func (id ZettelId) AppendBinary(text []byte) ([]byte, error) {
	return append(text, []byte(id.String())...), nil
}

func (id *ZettelId) UnmarshalBinary(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id ZettelId) ToType() TypeStruct {
	panic(errors.Err405MethodNotAllowed)
}

func (id ZettelId) ToSeq() doddish.Seq {
	return doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(id.left),
		},
		doddish.Token{
			Type:     doddish.TokenTypeOperator,
			Contents: []byte{'/'},
		},
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(id.right),
		},
	}
}

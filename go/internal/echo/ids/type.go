package ids

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

// TODO MAYBE move Type into its own package

func init() {
	register(typeStruct{})
}

type typeStruct struct {
	Value string
}

var (
	_ Type        = TypeStruct{}
	_ TypeMutable = &TypeStruct{}
)

func MakeTypeString(value string) string {
	value = strings.TrimSpace(value)

	if !strings.HasPrefix(value, "!") {
		value = "!" + value
	}

	return value
}

func MakeType(value string) (tipe SeqId, err error) {
	if err = tipe.SetType(value); err != nil {
		err = errors.Wrap(err)
		return tipe, err
	}

	if err = genres.Type.AssertGenre(tipe); err != nil {
		err = errors.Wrap(err)
		return tipe, err
	}

	return tipe, err
}

func MustType(value string) (tipe SeqId) {
	var err error
	tipe, err = MakeType(value)
	if err != nil {
		errors.PanicIfError(err)
	}

	return tipe
}

func MakeTypeStruct(value string) (tipe typeStruct, err error) {
	if err = tipe.Set(value); err != nil {
		err = errors.Wrap(err)
		return tipe, err
	}

	return tipe, err
}

func MustTypeStruct(value string) (tipe typeStruct) {
	if err := tipe.Set(value); err != nil {
		errors.PanicIfError(err)
	}

	return tipe
}

func (id typeStruct) IsEmpty() bool {
	return id.Value == ""
}

func (id *typeStruct) Reset() {
	id.Value = ""
}

func (id *typeStruct) ResetWith(b typeStruct) {
	id.Value = b.Value
}

func (id typeStruct) Equals(b typeStruct) bool {
	return id.Value == b.Value
}

func (id typeStruct) GetGenre() domain_interfaces.Genre {
	return genres.Type
}

func (id typeStruct) StringSansOp() string {
	if id.IsEmpty() {
		return ""
	} else {
		return id.Value
	}
}

func (id typeStruct) String() string {
	if id.IsEmpty() {
		return ""
	} else {
		return "!" + id.Value
	}
}

func (id *typeStruct) SetWithSeq(seq doddish.Seq) (err error) {
	var genre genres.Genre

	if genre, err = ValidateSeqAndGetGenre(seq); err != nil {
		err = errors.Wrap(err)
		return
	}

	if err = genre.AssertGenre(genres.Type); err != nil {
		err = errors.Wrap(err)
		return
	}

	id.Value = seq.String()

	return
}

func (id *typeStruct) Set(value string) (err error) {
	value = strings.ToLower(strings.TrimSpace(strings.Trim(value, ".! ")))

	if err = ErrOnConfig(value); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if !TagRegex.Match([]byte(value)) {
		err = errors.ErrorWithStackf("not a valid Type: '%s'", value)
		return err
	}

	id.Value = value

	return err
}

func (id typeStruct) MarshalText() (text []byte, err error) {
	text = []byte(id.String())
	return text, err
}

func (id *typeStruct) UnmarshalText(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id typeStruct) MarshalBinary() (text []byte, err error) {
	return id.AppendBinary(nil)
}

func (id typeStruct) AppendBinary(text []byte) ([]byte, error) {
	text = append(text, []byte(id.String())...)
	return text, nil
}

func (id *typeStruct) UnmarshalBinary(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id typeStruct) ToType() TypeStruct {
	return id
}

func (id typeStruct) ToSeq() doddish.Seq {
	return doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeOperator,
			Contents: []byte("!"),
		},
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(id.StringSansOp()),
		},
	}
}

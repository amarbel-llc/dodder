package ids

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

func MakeExternalObjectId(genre genres.Genre, value string) *ExternalObjectId {
	return &ExternalObjectId{
		value: value,
		genre: genre,
	}
}

type ExternalObjectId struct {
	value string
	genre genres.Genre
}

var _ Id = &ExternalObjectId{}

func (id *ExternalObjectId) GetExternalObjectId() domain_interfaces.ExternalObjectId {
	return id
}

func (id *ExternalObjectId) GetGenre() domain_interfaces.Genre {
	return id.genre
}

func (id *ExternalObjectId) IsEmpty() bool {
	return id.value == ""
}

func (id *ExternalObjectId) String() string {
	return id.value
}

func (id *ExternalObjectId) SetGenre(genre domain_interfaces.Genre) (err error) {
	id.genre = genres.Must(genre)
	return err
}

func (id *ExternalObjectId) SetBlob(v string) (err error) {
	id.genre = genres.Blob

	if err = id.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *ExternalObjectId) Set(value string) (err error) {
	if value == "/" {
		id.Reset()
		return err
	}

	if len(value) <= 1 {
		err = errors.ErrorWithStackf("external object id must be at least two characters, but got %q", value)
		return err
	}

	id.value = value

	return err
}

func (id *ExternalObjectId) SetWithGenre(
	value string,
	genre domain_interfaces.Genre,
) (err error) {
	if err = id.Set(value); err != nil {
		err = errors.Wrap(err)
		return err
	}

	id.genre = genres.Must(genre)
	return err
}

func (id *ExternalObjectId) Reset() {
	id.genre = genres.Unknown
	id.value = ""
}

func (id *ExternalObjectId) ResetWith(src *ExternalObjectId) {
	id.genre = src.genre
	id.value = src.value
}

func (id *ExternalObjectId) SetObjectIdLike(other Id) (err error) {
	if other.IsEmpty() {
		id.Reset()
		return err
	}

	otherGenre := genres.Must(other.GetGenre())

	if otherGenre.IsType() {
		if err = id.SetWithGenre(
			other.ToType().StringSansOp(),
			genres.Must(other.GetGenre()),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = id.SetWithGenre(
			other.String(),
			genres.Must(other.GetGenre()),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (id *ExternalObjectId) MarshalBinary() (bites []byte, err error) {
	if bites, err = id.genre.MarshalBinary(); err != nil {
		err = errors.Wrap(err)
		return bites, err
	}

	bites = append(bites, []byte(id.value)...)

	return bites, err
}

func (id *ExternalObjectId) AppendBinary(bites []byte) ([]byte, error) {
	{
		var err error

		if bites, err = id.genre.AppendBinary(bites); err != nil {
			err = errors.Wrap(err)
			return bites, err
		}
	}

	bites = append(bites, []byte(id.value)...)

	return bites, nil
}

func (id *ExternalObjectId) UnmarshalBinary(bites []byte) (err error) {
	if err = id.genre.UnmarshalBinary(bites[:1]); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if len(bites) > 1 {
		id.value = string(bites[1:])
	}

	return err
}

func (id *ExternalObjectId) ToType() TypeStruct {
	errors.PanicIfError(genres.Type.AssertGenre(id.genre))

	return TypeStruct{
		Value: id.value,
	}
}

func (id *ExternalObjectId) ToSeq() doddish.Seq {
	panic(errors.Err501NotImplemented)
}

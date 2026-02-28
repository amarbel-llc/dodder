package ids

import (
	"regexp"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/doddish"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

const RepoIdRegexString = `^(/)?[-a-z0-9_]+$`

var RepoIdRegex *regexp.Regexp

func init() {
	RepoIdRegex = regexp.MustCompile(RepoIdRegexString)
	register(RepoId{})
}

func MustRepoId(v string) (e *RepoId) {
	e = &RepoId{}

	if err := e.Set(v); err != nil {
		errors.PanicIfError(err)
	}

	return e
}

func MakeRepoId(v string) (e *RepoId, err error) {
	e = &RepoId{}

	if err = e.Set(v); err != nil {
		err = errors.Wrap(err)
		return e, err
	}

	return e, err
}

type RepoId struct {
	domain string
	id     string
}

func (id RepoId) IsEmpty() bool {
	return id.id == ""
}

func (id RepoId) GetRepoId() domain_interfaces.RepoId {
	return id
}

func (id RepoId) EqualsRepoId(kg domain_interfaces.RepoIdGetter) bool {
	return kg.GetRepoId().GetRepoIdString() == id.GetRepoIdString()
}

func (id RepoId) GetRepoIdString() string {
	return id.String()
}

func (id *RepoId) Reset() {
	id.id = ""
}

func (id *RepoId) ResetWith(e1 RepoId) {
	id.id = e1.id
}

func (id RepoId) Equals(b RepoId) bool {
	return id.id == b.id
}

func (id RepoId) GetGenre() domain_interfaces.Genre {
	return genres.Repo
}

func (id RepoId) GetObjectIdString() string {
	return id.String()
}

func (id RepoId) String() string {
	return id.id
}

func (id RepoId) StringWithSlashPrefix() string {
	return "/" + id.id
}

func (id RepoId) GetQueryPrefix() string {
	return "/"
}

func (id *RepoId) Set(v string) (err error) {
	v = strings.TrimPrefix(v, "/")
	v = strings.ToLower(strings.TrimSpace(v))

	if err = ErrOnConfig(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if v == "" {
		return err
	}

	if !RepoIdRegex.Match([]byte(v)) {
		err = errors.ErrorWithStackf("not a valid Kasten: '%s'", v)
		return err
	}

	id.id = v

	return err
}

func (id RepoId) MarshalText() (text []byte, err error) {
	text = []byte(id.String())
	return text, err
}

func (id *RepoId) UnmarshalText(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id RepoId) MarshalBinary() (text []byte, err error) {
	return id.AppendBinary(nil)
}

func (id RepoId) AppendBinary(text []byte) ([]byte, error) {
	return append(text, []byte(id.String())...), nil
}

func (id *RepoId) UnmarshalBinary(text []byte) (err error) {
	if err = id.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id RepoId) ToType() TypeStruct {
	panic(errors.Err405MethodNotAllowed)
}

func (id RepoId) ToSeq() doddish.Seq {
	return doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeOperator,
			Contents: []byte{'/'},
		},
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: []byte(id.id),
		},
	}
}

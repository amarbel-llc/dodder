package ids

import (
	"bytes"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

func init() {
	register(config{})
}

var configBytes = []byte("konfig")

// TODO move to doddish
func TokenIsConfig(token doddish.Token) bool {
	return bytes.Equal(token.Contents, configBytes)
}

func ErrOnConfigBytes(b []byte) (err error) {
	if bytes.Equal(b, configBytes) {
		return errors.ErrorWithStackf("cannot be %q", "konfig")
	}

	return nil
}

func ErrOnConfig(value string) (err error) {
	if value == "konfig" {
		return errors.ErrorWithStackf("cannot be %q", "konfig")
	}

	return nil
}

type config struct{}

var Config config

func (config config) IsEmpty() bool {
	return false
}

func (config config) GetGenre() domain_interfaces.Genre {
	return genres.Config
}

func (config *config) Reset() {
}

func (config *config) ResetWith(_ config) {
}

func (config config) GetObjectIdString() string {
	return config.String()
}

func (config config) String() string {
	return "konfig"
}

func (config config) Set(value string) (err error) {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	if value != "konfig" {
		err = errors.Errorf("not konfig")
		return err
	}

	return err
}

func (config config) MarshalText() (text []byte, err error) {
	text = []byte(config.String())
	return text, err
}

func (config *config) UnmarshalText(text []byte) (err error) {
	if err = config.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (config config) MarshalBinary() (text []byte, err error) {
	return config.AppendBinary(nil)
}

func (config config) AppendBinary(text []byte) ([]byte, error) {
	return append(text, []byte(config.String())...), nil
}

func (config *config) UnmarshalBinary(text []byte) (err error) {
	if err = config.Set(string(text)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (config config) ToType() TypeStruct {
	panic(errors.Err405MethodNotAllowed)
}

func (config config) ToSeq() doddish.Seq {
	return doddish.Seq{
		doddish.Token{
			Type:     doddish.TokenTypeIdentifier,
			Contents: configBytes,
		},
	}
}

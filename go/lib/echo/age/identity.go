package age

import (
	"bufio"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
	"filippo.io/age"
)

// necessary because the age.Identity interface does not include Stringer, but
// all of the actual identities do implement Stringer
type identity interface {
	age.Identity
	interfaces.Stringer
}

type Identity struct {
	identity identity
	age.Recipient

	path     string
	disabled bool
}

func (identity Identity) Unwrap(
	stanzas []*age.Stanza,
) (fileKey []byte, err error) {
	if fileKey, err = identity.identity.Unwrap(stanzas); err != nil {
		err = errors.Wrap(err)
		return fileKey, err
	}

	return fileKey, err
}

func (identity *Identity) IsDisabled() bool {
	return identity.disabled
}

func (identity *Identity) IsEmpty() bool {
	return identity.identity == nil
}

func (identity *Identity) String() string {
	if identity.identity == nil {
		return ""
	} else {
		return identity.identity.String()
	}
}

func (identity *Identity) MarshalText() (b []byte, err error) {
	b = []byte(identity.String())
	return b, err
}

func (identity *Identity) UnmarshalText(b []byte) (err error) {
	if err = identity.SetFromX25519Identity(string(b)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (identity *Identity) SetFromX25519Identity(
	identityString string,
) (err error) {
	var x *age.X25519Identity

	if x, err = age.ParseX25519Identity(identityString); err != nil {
		err = errors.Wrapf(err, "Identity: %s", identityString)
		return err
	}

	identity.SetX25519Identity(x)

	return err
}

func (identity *Identity) SetX25519Identity(x *age.X25519Identity) {
	identity.disabled = false
	identity.identity = x
	identity.Recipient = x.Recipient()
}

func (identity *Identity) SetFromPath(path string) (err error) {
	var f *os.File

	if f, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, f)

	br := bufio.NewReader(f)
	isEOF := false
	var key string

	for !isEOF {
		var line string
		line, err = br.ReadString('\n')

		if err == io.EOF {
			isEOF = true
			err = nil
		} else if err != nil {
			err = errors.Wrap(err)
			return err
		}

		if len(line) > 0 {
			key = strings.TrimSpace(line)
		}
	}

	if err = identity.SetFromX25519Identity(key); err != nil {
		err = errors.Wrapf(err, "Key: %q", key)
		return err
	}

	return err
}

func (identity *Identity) Set(path_or_identity string) (err error) {
	switch {
	case path_or_identity == "":

	case path_or_identity == "disabled" || path_or_identity == "none":
		identity.disabled = true
		// no-op

	case files.Exists(path_or_identity):
		if err = identity.SetFromPath(path_or_identity); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case path_or_identity == "generate":
		if err = identity.GenerateIfNecessary(); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		if err = identity.SetFromX25519Identity(path_or_identity); err != nil {
			err = errors.Wrapf(err, "Identity: %q", path_or_identity)
			return err
		}
	}

	return err
}

func (identity *Identity) GenerateIfNecessary() (err error) {
	if identity.IsDisabled() || !identity.IsEmpty() {
		return err
	}

	var x *age.X25519Identity

	if x, err = age.GenerateX25519Identity(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	identity.SetX25519Identity(x)

	return err
}

func (identity *Identity) WrapReader(
	src io.Reader,
) (out io.ReadCloser, err error) {
	if src, err = age.Decrypt(src, identity); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	out = ohio.NopCloser(src)

	return out, err
}

func (identity *Identity) WrapWriter(
	dst io.Writer,
) (out io.WriteCloser, err error) {
	if out, err = age.Encrypt(dst, identity.Recipient); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	return out, err
}

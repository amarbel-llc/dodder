package age

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"filippo.io/age"
)

type MultiIdentity struct {
	identities []Identity
}

func MakeMultiIdentity(identities []Identity) MultiIdentity {
	return MultiIdentity{identities: identities}
}

func (mi *MultiIdentity) WrapReader(
	src io.Reader,
) (out io.ReadCloser, err error) {
	if len(mi.identities) == 0 {
		out = ohio.NopCloser(src)
		return out, err
	}

	ageIdentities := make([]age.Identity, len(mi.identities))

	for i := range mi.identities {
		ageIdentities[i] = &mi.identities[i]
	}

	var reader io.Reader

	if reader, err = age.Decrypt(src, ageIdentities...); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	out = ohio.NopCloser(reader)

	return out, err
}

func (mi *MultiIdentity) WrapWriter(
	dst io.Writer,
) (out io.WriteCloser, err error) {
	if len(mi.identities) == 0 {
		out = ohio.NopWriteCloser(dst)
		return out, err
	}

	recipients := make([]age.Recipient, len(mi.identities))

	for i := range mi.identities {
		recipients[i] = mi.identities[i].Recipient
	}

	if out, err = age.Encrypt(dst, recipients...); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	return out, err
}

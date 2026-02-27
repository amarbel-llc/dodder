package markl_age_id

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/lib/_/bech32"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/echo/age"
	age_upstream "filippo.io/age"
)

type (
	X25519Identity  = age_upstream.X25519Identity
	X25519Recipient = age_upstream.X25519Recipient
)

type Id struct {
	// Recipients []Recipient `toml:"recipients,omitempty"`
	Identities []*age.Identity `toml:"identities,omitempty"`
}

var _ domain_interfaces.MarklId = Id{}

func (id Id) String() string {
	if len(id.Identities) == 0 {
		return ""
	} else {
		return fmt.Sprintf("%s", id.Identities)
	}
}

func (id *Id) Set(v string) (err error) {
	var identity age.Identity

	if err = identity.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.AddIdentity(identity); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (id *Id) AddIdentity(
	identity age.Identity,
) (err error) {
	if identity.IsDisabled() || identity.IsEmpty() {
		return err
	}

	// a.Recipients = append(a.Recipients, identity)
	id.Identities = append(id.Identities, &identity)

	return err
}

func (id *Id) AddIdentityOrGenerateIfNecessary(
	identity age.Identity,
) (err error) {
	if err = identity.GenerateIfNecessary(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = id.AddIdentity(identity); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func MakeFromIdentity(identity age.Identity) (a *Id, err error) {
	a = &Id{}
	err = a.AddIdentityOrGenerateIfNecessary(identity)
	return a, err
}

func MakeFromIdentityPathOrString(path_or_identity string) (a *Id, err error) {
	var i age.Identity

	if err = i.Set(path_or_identity); err != nil {
		err = errors.Wrap(err)
		return a, err
	}

	return MakeFromIdentity(i)
}

func MakeFromIdentityFile(basePath string) (a *Id, err error) {
	var i age.Identity

	if err = i.SetFromPath(basePath); err != nil {
		err = errors.Wrap(err)
		return a, err
	}

	return MakeFromIdentity(i)
}

func MakeFromIdentityString(contents string) (a *Id, err error) {
	var i age.Identity

	if err = i.SetFromX25519Identity(contents); err != nil {
		err = errors.Wrap(err)
		return a, err
	}

	return MakeFromIdentity(i)
}

// func (a *Age) AddBech32PivYubikeyEC256(bech string) (err error) {
// 	var r *age.PivYubikeyEC256Recipient

// 	if r, err = age.ParseBech32PivYubikeyEC256Recipient(bech); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	var i *age.PivYubikeyEC256Identity

// 	if i, err = age.ReadPivYubikeyEC256Identity(r); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	a.recipients = append(a.recipients, r)
// 	a.identities = append(a.identities, i)

// 	return
// }

func (id *Id) GetRecipients() []age_upstream.Recipient {
	r := make([]age_upstream.Recipient, len(id.Identities))

	for i := range r {
		r[i] = id.Identities[i]
	}

	return r
}

func (id Id) StringWithFormat() string {
	return id.String()
}

func (id Id) MarshalBinary() ([]byte, error) {
	return nil, errors.Err405MethodNotAllowed
}

func (id Id) GetBytes() []byte {
	switch len(id.Identities) {
	default:
		panic(
			errors.Errorf(
				"unsupported age state for getting bytes: >1 identity",
			),
		)

	case 0:
		return nil

	case 1:
		_, data, err := bech32.Decode(id.Identities[0].String())
		errors.PanicIfError(err)
		return data
	}
}

func (id Id) GetSize() int {
	panic(errors.Err405MethodNotAllowed)
}

func (id Id) GetMarklFormat() domain_interfaces.MarklFormat {
	if id.IsNull() {
		return nil
	} else {
		return tipe{}
	}
}

func (id Id) IsEmpty() bool {
	return len(id.Identities) == 0
}

func (id Id) IsNull() bool {
	return len(id.Identities) == 0
}

func (id Id) GetPurposeId() string {
	return markl.PurposeMadderPrivateKeyV1
}

func (id Id) GetIOWrapper() (ioWrapper interfaces.IOWrapper, err error) {
	if id.IsNull() {
		ioWrapper = ohio.NopeIOWrapper{}
		return ioWrapper, err
	}

	var formatSec markl.FormatSec

	if formatSec, err = markl.GetFormatSecOrError(
		markl.FormatId(markl.FormatIdAgeX25519Sec),
	); err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	if formatSec.GetIOWrapper == nil {
		err = errors.Errorf(
			"format does not support getting io wrapper key: %q",
			formatSec.GetMarklFormatId(),
		)
		return ioWrapper, err
	}

	if ioWrapper, err = formatSec.GetIOWrapper(id); err != nil {
		err = errors.Wrap(err)
		return ioWrapper, err
	}

	return ioWrapper, err
}

func (id Id) Verify(_, _ domain_interfaces.MarklId) (err error) {
	return errors.Err405MethodNotAllowed
}

func (id Id) Sign(
	mes domain_interfaces.MarklId,
	sigDst domain_interfaces.MarklIdMutable,
	sigPurpose string,
) (err error) {
	return errors.Err405MethodNotAllowed
}

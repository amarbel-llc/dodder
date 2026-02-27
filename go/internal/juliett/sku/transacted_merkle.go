package sku

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_fmt_digest"
)

func (transacted *Transacted) SetMother(mother *Transacted) (err error) {
	motherSig := transacted.GetMetadataMutable().GetMotherObjectSigMutable()

	if mother == nil {
		motherSig.Reset()
		return err
	}

	if err = motherSig.SetMarklId(
		markl.FormatIdEd25519Sig,
		mother.GetMetadata().GetObjectSig().GetBytes(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = motherSig.SetPurposeId(
		markl.GetMotherSigTypeForSigType(
			mother.GetMetadata().GetObjectSig().GetPurposeId(),
		),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) AssertObjectDigestAndObjectSigNotNull() (err error) {
	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadata().GetObjectDigest()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadata().GetObjectSig()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) Verify() (err error) {
	pubKey := transacted.GetMetadata().GetRepoPubKey()

	if err = markl.AssertIdIsNotNull(
		pubKey,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadata().GetObjectDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadata().GetObjectSig(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = pubKey.Verify(
		transacted.GetMetadata().GetObjectDigest(),
		transacted.GetMetadata().GetObjectSig(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) CalculateObjectDigest(
	defaultObjectDigestPurposeId string,
) (err error) {
	if err = transacted.CalculateDigestForPurpose(
		defaultObjectDigestPurposeId,
		transacted.GetMetadataMutable().GetObjectDigestMutable(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) CalculateDigestForPurpose(
	purposeId string,
	digest domain_interfaces.MarklIdMutable,
) (err error) {
	if err = object_fmt_digest.WriteDigest(
		purposeId,
		transacted,
		digest,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

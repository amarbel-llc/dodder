package sku

import (
	"code.linenisgreat.com/dodder/go/internal/echo/object_fmt_digest"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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

	return err
}

// ObjectDigestPurposeOrDefault returns the digest purpose to use when
// recomputing this object's digest for verification: derived from the
// object's own signature purpose (the markl "digest" relation), so
// objects signed under different sig versions (v2/v3) coexist within
// one repo. Unsigned objects fall back to defaultPurposeId (typically
// the repo-global envRepo.GetObjectDigestType()).
func (transacted *Transacted) ObjectDigestPurposeOrDefault(
	defaultPurposeId string,
) string {
	sig := transacted.GetMetadata().GetObjectSig()

	if sig.IsNull() || sig.GetPurposeId() == "" {
		return defaultPurposeId
	}

	return markl.GetDigestTypeForSigType(sig.GetPurposeId())
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
	digest mad_domain_interfaces.MarklIdMutable,
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

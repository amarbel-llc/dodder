package object_finalizer

import (
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (finalizer *Finalizer) ValidateIfNecessary(
	daughter *sku.Transacted,
	mother *sku.Transacted,
	options sku.CommitOptions,
	// typedBlobStores typed_blob_store.Stores,
) (err error) {
	if !options.Validate {
		return err
	}

	switch daughter.GetSku().GetGenre() {
	case genres.Type:
		// var repool interfaces.FuncRepool

		// if _, repool, _, err = typedBlobStores.Type.ParseTypedBlob(
		// 	daughter.GetType(),
		// 	daughter.GetSku().GetBlobDigest(),
		// ); err != nil {
		// 	err = errors.Wrap(err)
		// 	return err
		// }

		// defer repool()
	}

	return err
}

type VerifyOptions struct {
	PubKeyPresent       bool
	ObjectDigestPresent bool
	ObjectSigPresent    bool
	ObjectSigValid      bool
}

var defaultVerifyOptions = VerifyOptions{
	PubKeyPresent:       true,
	ObjectDigestPresent: true,
	ObjectSigPresent:    true,
	ObjectSigValid:      true,
}

func DefaultVerifyOptions() VerifyOptions {
	return defaultVerifyOptions
}

func (finalizer *Finalizer) Verify(
	transacted *sku.Transacted,
) (err error) {
	return finalizer.verify(transacted, finalizer.verifyOptions)
}

func (finalizer *Finalizer) verify(
	transacted *sku.Transacted,
	options VerifyOptions,
) (err error) {
	pubKey := transacted.GetMetadata().GetRepoPubKey()

	if options.PubKeyPresent {
		if err = markl.AssertIdIsNotNullWithPurpose(
			pubKey,
			"pubkey",
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if options.ObjectDigestPresent {
		if err = markl.AssertIdIsNotNullWithPurpose(
			transacted.GetMetadata().GetObjectDigest(),
			"object-dig",
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if options.ObjectSigPresent {
		if err = markl.AssertIdIsNotNullWithPurpose(
			transacted.GetMetadata().GetObjectSig(),
			"object-sig",
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if options.PubKeyPresent &&
		options.ObjectSigValid &&
		options.ObjectSigPresent &&
		options.ObjectDigestPresent {
		if err = pubKey.Verify(
			transacted.GetMetadata().GetObjectDigest(),
			transacted.GetMetadata().GetObjectSig(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

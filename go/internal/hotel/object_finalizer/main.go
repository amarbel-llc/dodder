package object_finalizer

import (
	"slices"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type (
	FinalizerGetter interface {
		GetObjectFinalizer() Finalizer
	}

	Finalizer = finalizer

	finalizer struct {
		index         sku.IndexPrimitives
		verifyOptions VerifyOptions
		// pubKey interfaces.MarklId
	}

	object = *sku.Transacted

	// object interface {
	// 	object_metadata.GetterMutable

	// 	CalculateDigests(
	// 		debug bool,
	// 		formats interfaces.DigestWriteMap,
	// 	) (err error)

	// 	GetDigestWriteMapWithMerkle() interfaces.DigestWriteMap
	// 	GetDigestWriteMapWithoutMerkle() interfaces.DigestWriteMap
	// 	Verify() (err error)
	// }
)

func Make() Finalizer {
	return Builder().Build()
}

// func Make(index sku.IndexPrimitives) Finalizer {
// 	return finalizer{index: index}
// }

func (finalizer finalizer) GetObjectFinalizer() Finalizer {
	return finalizer
}

// TODO extract into a versioned object finalizer
// calculates the object digests using the object's repo pubkey
func (finalizer finalizer) FinalizeUsingObject(
	transacted object,
	objectDigestMarklFormatId string,
) (err error) {
	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadataMutable().GetRepoPubKey(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = finalizer.FinalizeUsingRepoPubKey(
		transacted,
		transacted.GetMetadataMutable().GetRepoPubKey(),
		objectDigestMarklFormatId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// calculates the object digests using the provided repo pubkey
func (finalizer finalizer) FinalizeUsingRepoPubKey(
	object object,
	pubKey domain_interfaces.MarklId,
	objectDigestMarklFormatId string,
) (err error) {
	metadataMutable := object.GetMetadataMutable()
	// TODO migrate this to config
	pubKeyMutable := metadataMutable.GetRepoPubKeyMutable()

	if pubKeyMutable.IsNull() {
		pubKeyMutable.ResetWithMarklId(pubKey)
		// if err = markl.SetMerkleIdWithFormat(
		// 	transacted.GetMetadataMutable().GetRepoPubKeyMutable(),
		// 	markl.FormatIdRepoPubKeyV1,
		// 	pubKey,
		// ); err != nil {
		// 	err = errors.Wrap(err)
		// 	return
		// }
	} else {
		if err = markl.MakeErrNotEqualBytes(
			pubKey.GetBytes(),
			pubKeyMutable.GetBytes(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = finalizer.WriteLockfileIfNecessary(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = object.CalculateObjectDigest(
		objectDigestMarklFormatId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (finalizer finalizer) WriteLockfileIfNecessary(
	object object,
) (err error) {
	if finalizer.index == nil {
		return err
	}

	return finalizer.WriteLockfile(
		object,
		sku.LockfileOptions{},
		finalizer.index.ReadOneMarklIdAdded,
		finalizer.index.ReadOneMarklId,
	)
}

// TODO offer option to allow type lock failures
func (finalizer finalizer) WriteLockfile(
	object object,
	options sku.LockfileOptions,
	funcs ...sku.FuncReadOne,
) (err error) {
	metadata := object.GetMetadataMutable()

	tipe := metadata.GetType()

	if err = finalizer.writeTypeLockIfNecessary(
		metadata,
		tipe,
		funcs...,
	); err != nil {
		switch err {
		case ErrEmptyLockKey, ErrBuiltinType:
			err = nil

		case ErrFailedToReadCurrentLockObject:
			if options.AllowTypeFailure {
				err = nil
				break
			}

			fallthrough

		default:
			err = errors.Wrapf(err, "failed to write type lock for type: %q", tipe)
			return err
		}
	}

	for tag := range metadata.GetTags().All() {
		if err = finalizer.writeTagLockIfNecessary(
			metadata,
			tag,
			funcs...,
		); err != nil {
			switch err {
			case ErrEmptyLockKey:
				err = nil

			case ErrFailedToReadCurrentLockObject:
				err = nil

			default:
				err = errors.Wrapf(err, "failed to write tag lock for tag: %q", tag)
				return err
			}
		}
	}

	return err
}

func (finalizer finalizer) FinalizeAndSignIfNecessary(
	transacted object,
	config genesis_configs.ConfigPrivate,
) (err error) {
	if !transacted.GetMetadataMutable().GetObjectSig().IsNull() {
		return err
	}

	if err = finalizer.FinalizeAndSign(transacted, config); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if transacted.GetMetadataMutable().GetRepoPubKey().GetPurposeId() == "" {
		panic("empty pubkey format")
	}

	return err
}

func (finalizer finalizer) FinalizeAndSignOverwrite(
	transacted object,
	config genesis_configs.ConfigPrivate,
) (err error) {
	// TODO populate format ids from config
	transacted.GetMetadataMutable().GetObjectSigMutable().Reset()
	transacted.GetMetadataMutable().GetRepoPubKeyMutable().Reset()

	if err = finalizer.FinalizeAndSign(
		transacted,
		config,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (finalizer finalizer) FinalizeAndSign(
	transacted object,
	config genesis_configs.ConfigPrivate,
) (err error) {
	if err = markl.AssertIdIsNull(
		transacted.GetMetadataMutable().GetRepoPubKey(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNull(
		transacted.GetMetadataMutable().GetObjectSig(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	transacted.GetMetadataMutable().GetRepoPubKeyMutable().ResetWithMarklId(
		config.GetPublicKey(),
	)

	if err = finalizer.FinalizeUsingObject(
		transacted,
		markl.GetDigestTypeForSigType(config.GetObjectSigMarklTypeId()),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertIdIsNotNull(
		transacted.GetMetadataMutable().GetObjectDigest()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	privateKey := config.GetPrivateKey()

	if err = privateKey.Sign(
		transacted.GetMetadataMutable().GetObjectDigest(),
		transacted.GetMetadataMutable().GetObjectSigMutable(),
		config.GetObjectSigMarklTypeId(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = finalizer.Verify(transacted); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (finalizer finalizer) FinalizeAndVerify(
	transacted object,
	objectDigestMarklFormatId string,
) (err error) {
	if err = finalizer.FinalizeUsingObject(
		transacted,
		objectDigestMarklFormatId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if slices.Contains(
		[]string{ids.TypeInventoryListV1},
		transacted.GetMetadataMutable().GetType().String(),
	) {
		return err
	}

	if err = finalizer.Verify(transacted); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

package object_finalizer

import (
	"fmt"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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
		// TEMPORARY debug instrumentation for investigating the pull
		// ed25519-verification bug: logs the exact pubkey/digest/sig bytes
		// being compared for a single object-id, gated behind an env var
		// so it's a no-op otherwise. Not meant to be committed long-term
		// -- remove once the bug is root-caused.
		debugObjectIdSubstr := os.Getenv("DODDER_DEBUG_DIGEST_OBJECT_ID")
		if debugObjectIdSubstr != "" &&
			strings.Contains(transacted.GetObjectId().String(), debugObjectIdSubstr) {
			digest := transacted.GetMetadata().GetObjectDigest()
			sig := transacted.GetMetadata().GetObjectSig()

			fmt.Fprintf(
				os.Stderr,
				"DODDER_DEBUG_VERIFY: object_id=%q\n  pubkey=%s\n  pubkey_bytes=%x\n  digest=%s\n  digest_bytes=%x\n  sig=%s\n  sig_bytes=%x\n",
				transacted.GetObjectId().String(),
				pubKey,
				pubKey.GetBytes(),
				digest,
				digest.GetBytes(),
				sig,
				sig.GetBytes(),
			)
		}

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

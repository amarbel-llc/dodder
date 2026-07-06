// Package markl_registrations holds dodder's own purpose
// registrations for the markl framework (madder#255 ownership model:
// each consumer registers the purposes it speaks). Since madder
// 69e4fa6 dropped its transitional dodder-* set, this package is the
// sole registration site for the dodder-* purposes.
//
// To activate, blank-import this package. The canonical activation
// site is internal/0/domain_interfaces (the lowest dodder-internal
// tier every downstream caller transitively imports), which keeps
// production binaries and unit-test binaries in sync.
//
// DUPLICATE HAZARD: markl.RegisterPurpose and
// markl.RegisterPurposeIdAlias panic on duplicate ids. Madder's
// registrations (blank-imported below) own the madder-* purposes and
// the legacy zit/dodder private-key purpose-id aliases — dodder
// inherits those and must not register its own copies.
package markl_registrations

import (
	// Madder's registrations are madder-only since madder 69e4fa6:
	// the madder-* purposes dodder mints too (gen, markl_age_id) and
	// the legacy zit/dodder-repo-private_key-v1 → ed25519_sec
	// purpose-id aliases for pre-rename on-disk key files (madder#167).
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
	// The age and agent blank imports swap the real age/pivy-backed
	// implementations over piggy's erroring core stubs at their inits
	// (idempotent). The SSH signing formats stay stubs here —
	// connecting a signer is a consumer-side call (see
	// genesis_config_blobs' agent.RegisterSSHEd25519Format usage).
	_ "github.com/amarbel-llc/piggy/go/pkgs/age"
	_ "github.com/amarbel-llc/piggy/go/pkgs/agent"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"

	// Core format registrations plus the piggy-* purposes.
	_ "github.com/amarbel-llc/piggy/go/pkgs/markl_registrations"
)

// The dodder-owned purpose registrations, copied opt-for-opt from the
// transitional set madder carried through f692c55 and dropped in
// 69e4fa6 (go/internal/charlie/markl_registrations).

var (
	purposeBlobDigestV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeBlobDigestV1,
		Type: markl.PurposeTypeBlobDigest,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	}

	purposeObjectDigestV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectDigestV1,
		Type: markl.PurposeTypeObjectDigest,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	}

	purposeObjectDigestV2Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectDigestV2,
		Type: markl.PurposeTypeObjectDigest,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	}

	// v3 object digest/sig purposes: the signed object digest gains
	// typed blob-reference coverage (madder#255). Registration shape is
	// identical to v2 — the version bump changes what the digest
	// covers, not the formats.
	purposeObjectDigestV3Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectDigestV3,
		Type: markl.PurposeTypeObjectDigest,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	}

	purposeV5MetadataDigestWithoutTaiOpts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeV5MetadataDigestWithoutTai,
		Type: markl.PurposeTypeObjectDigest,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	}

	purposeObjectMotherSigV1Opts = markl.RegisterPurposeOpts{
		Id:        markl.PurposeObjectMotherSigV1,
		Type:      markl.PurposeTypeObjectMotherSig,
		FormatIds: []string{markl.FormatIdEd25519Sig},
	}

	purposeObjectMotherSigV2Opts = markl.RegisterPurposeOpts{
		Id:        markl.PurposeObjectMotherSigV2,
		Type:      markl.PurposeTypeObjectMotherSig,
		FormatIds: []string{markl.FormatIdEd25519Sig},
	}

	purposeObjectMotherSigV3Opts = markl.RegisterPurposeOpts{
		Id:        markl.PurposeObjectMotherSigV3,
		Type:      markl.PurposeTypeObjectMotherSig,
		FormatIds: []string{markl.FormatIdEd25519Sig},
	}

	purposeObjectSigV0Opts = markl.RegisterPurposeOpts{
		Id:        markl.PurposeObjectSigV0,
		Type:      markl.PurposeTypeDodderObjectSig,
		FormatIds: []string{markl.FormatIdEd25519Sig},
	}

	purposeObjectSigV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectSigV1,
		Type: markl.PurposeTypeDodderObjectSig,
		FormatIds: []string{
			markl.FormatIdEd25519Sig,
			markl.FormatIdEcdsaP256Sig,
		},
		Related: map[string]string{
			markl.RelatedRoleDigest:    markl.PurposeObjectDigestV1,
			markl.RelatedRoleMotherSig: markl.PurposeObjectMotherSigV1,
		},
	}

	purposeObjectSigV2Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectSigV2,
		Type: markl.PurposeTypeDodderObjectSig,
		FormatIds: []string{
			markl.FormatIdEd25519Sig,
			markl.FormatIdEcdsaP256Sig,
		},
		Related: map[string]string{
			markl.RelatedRoleDigest:    markl.PurposeObjectDigestV2,
			markl.RelatedRoleMotherSig: markl.PurposeObjectMotherSigV2,
		},
	}

	purposeObjectSigV3Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeObjectSigV3,
		Type: markl.PurposeTypeDodderObjectSig,
		FormatIds: []string{
			markl.FormatIdEd25519Sig,
			markl.FormatIdEcdsaP256Sig,
		},
		Related: map[string]string{
			markl.RelatedRoleDigest:    markl.PurposeObjectDigestV3,
			markl.RelatedRoleMotherSig: markl.PurposeObjectMotherSigV3,
		},
	}

	purposeRepoPrivateKeyV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeRepoPrivateKeyV1,
		Type: markl.PurposeTypePrivateKey,
		FormatIds: []string{
			markl.FormatIdEd25519Sec,
			markl.FormatIdEd25519SSH,
			markl.FormatIdEcdsaP256SSH,
		},
		Related: map[string]string{
			markl.RelatedRolePublicKey: markl.PurposeRepoPubKeyV1,
		},
	}

	purposeRepoPubKeyV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeRepoPubKeyV1,
		Type: markl.PurposeTypeRepoPubKey,
		FormatIds: []string{
			markl.FormatIdEd25519Pub,
			markl.FormatIdEcdsaP256Pub,
		},
	}

	purposeRequestAuthChallengeV1Opts = markl.RegisterPurposeOpts{
		Id:        markl.PurposeRequestAuthChallengeV1,
		Type:      markl.PurposeTypeRequestAuth,
		FormatIds: []string{markl.FormatIdNonceSec},
	}

	purposeRequestAuthResponseV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeRequestAuthResponseV1,
		Type: markl.PurposeTypeRequestAuth,
		FormatIds: []string{
			markl.FormatIdEd25519Sig,
			markl.FormatIdEcdsaP256Sig,
		},
	}

	purposeRequestRepoSigV1Opts = markl.RegisterPurposeOpts{
		Id:   markl.PurposeRequestRepoSigV1,
		Type: markl.PurposeTypeRequestAuth,
		FormatIds: []string{
			markl.FormatIdEd25519Sig,
			markl.FormatIdEcdsaP256Sig,
		},
	}
)

// AllPurposes is the ordered list of purposes dodder registers. Order
// is deterministic but nothing may depend on it — registration is
// order-independent under markl's lazy Related validation. Exported
// for the package's registration tests (mirroring madder's shape).
var AllPurposes = []markl.RegisterPurposeOpts{
	purposeBlobDigestV1Opts,
	purposeObjectDigestV1Opts,
	purposeObjectDigestV2Opts,
	purposeObjectDigestV3Opts,
	purposeV5MetadataDigestWithoutTaiOpts,
	purposeObjectMotherSigV1Opts,
	purposeObjectMotherSigV2Opts,
	purposeObjectMotherSigV3Opts,
	purposeObjectSigV0Opts,
	purposeObjectSigV1Opts,
	purposeObjectSigV2Opts,
	purposeObjectSigV3Opts,
	purposeRepoPrivateKeyV1Opts,
	purposeRepoPubKeyV1Opts,
	purposeRequestAuthChallengeV1Opts,
	purposeRequestAuthResponseV1Opts,
	purposeRequestRepoSigV1Opts,
}

// The legacy zit/dodder-repo-private_key-v1 → ed25519_sec purpose-id
// aliases are NOT registered here: madder keeps them (its blob stores
// read the same pre-rename on-disk wire forms) and dodder inherits
// them via the blank import above. RegisterPurposeIdAlias panics on
// duplicates.

func init() {
	for _, opts := range AllPurposes {
		markl.RegisterPurpose(opts)
	}
}

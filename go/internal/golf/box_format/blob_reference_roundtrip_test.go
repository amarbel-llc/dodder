package box_format

import (
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// makeSignedObjectWithBlobReferences builds a signed type object shaped like
// the genesis pandoc !md: a builtin meta-type (no object type-lock value) and
// three aliased blob references, each carrying a type lock with a value (the
// shape genesis produces via addToolBlobReference + the lockfile pass). This
// is the object shape that fails signature verification after an
// inventory-list round-trip (repo-fsck / transfer / last). The object is
// signed under sigPurpose (PurposeObjectSigV2 or PurposeObjectSigV3).
func makeSignedObjectWithBlobReferences(
	t *ui.T,
	sigPurpose string,
) (*sku.Transacted, genesis_configs.ConfigPrivate) {
	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustTypeStruct("md")),
	)
	object.SetTai(ids.NowTai())

	metadata := object.GetMetadataMutable()

	t.AssertNoError(markl.SetHexBytes(
		markl.FormatIdHashSha256,
		metadata.GetBlobDigestMutable(),
		[]byte(
			"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
		),
	))

	t.AssertNoError(metadata.GetTypeMutable().SetType("!toml-type-v2"))

	// A multi-paragraph description (embedded blank-line break) is part
	// of this object's realistic shape and exercises the archive-format
	// newline-preservation path (box_format/transacted.go's isArchive
	// branch) in the same round-trip this test already verifies for blob
	// references -- without it, this regression test would never have
	// caught the description-newline corruption bug.
	t.AssertNoError(metadata.GetDescriptionMutable().Set(
		"first paragraph of the description.\n\nsecond paragraph, after a blank line.",
	))

	for _, ref := range []struct {
		blobHex    string
		typeString string
		alias      string
	}{
		{
			blobHex:    "11500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2411",
			typeString: "pandoc-lua_filter",
			alias:      "filters/dodder-common.lua",
		},
		{
			blobHex:    "22500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2422",
			typeString: "pandoc-lua_filter",
			alias:      "filters/dodder-edit.lua",
		},
		{
			blobHex:    "33500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2433",
			typeString: "pandoc-defaults",
			alias:      "defaults/dodder-edit.yaml",
		},
	} {
		var blobId markl.Id

		t.AssertNoError(markl.SetHexBytes(
			markl.FormatIdHashSha256,
			&blobId,
			[]byte(ref.blobHex),
		))

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

		t.AssertNoError(marshaler.Set(ids.MakeTypeString(ref.typeString)))

		// the lockfile pass stamps the referenced type object's sig as the
		// lock value; fake it with a generated key like
		// self_provenance_test does for the object type lock
		t.AssertNoError(typeLock.GetValueMutable().GeneratePrivateKey(
			nil,
			markl.FormatIdNonceSec,
			"",
		))

		metadata.AddBlobReference(blobId, typeLock)
		t.AssertNoError(metadata.SetBlobReferenceAlias(blobId, ref.alias))
	}

	config := genesis_configs.Default().Blob
	config.SetObjectSigMarklTypeId(sigPurpose)
	t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
		nil,
		markl.FormatIdEd25519Sec,
		markl.PurposeRepoPrivateKeyV1,
	))

	finalizer := object_finalizer.Make()
	t.AssertNoError(finalizer.FinalizeAndSignOverwrite(object, config))

	return object, config
}

// digestFields collects the string form of every field that feeds the object
// digest (object_fmt_digest PurposeObjectDigestV2: blob digest, description,
// object id + genre, tags, tai, type lock, repo pubkey, mother sig) plus the
// blob references, so a round-trip mismatch names the corrupted field.
func digestFields(object *sku.Transacted) map[string]string {
	metadata := object.GetMetadata()

	out := map[string]string{
		"blob-digest": metadata.GetBlobDigest().String(),
		"description": metadata.GetDescription().String(),
		"genre":       object.GetObjectId().GetGenre().String(),
		"object-id":   object.GetObjectId().String(),
		"tai":         metadata.GetTai().String(),
		"type":        metadata.GetType().String(),
		"type-lock": markl.MakeLockCoderValueNotRequired(
			metadata.GetTypeLock(),
		).String(),
		"repo-pub":   metadata.GetRepoPubKey().String(),
		"mother-sig": metadata.GetMotherObjectSig().String(),
	}

	var tags []string
	for tag := range metadata.AllTags() {
		tags = append(tags, tag.String())
	}
	out["tags"] = strings.Join(tags, ", ")

	var refs []string
	for blobId := range metadata.AllBlobReferences() {
		refs = append(
			refs,
			metadata.GetBlobReferenceAlias(blobId)+
				"<@"+blobId.String()+
				" "+markl.MakeLockCoderValueNotRequired(
				metadata.GetBlobReferenceTypeLock(blobId),
			).String(),
		)
	}
	out["blob-refs"] = strings.Join(refs, " | ")

	return out
}

// makeArchiveBoxForTest mirrors MakeBoxTransactedArchive's option set
// without requiring an env_ui.Env (only used there for color options, which
// the archive form turns off entirely).
func makeArchiveBoxForTest() *BoxTransacted {
	options := options_print.Options{}.
		WithPrintTai(true).
		WithPrintBlobDigests(true).
		WithExcludeFields(true).
		WithDescriptionInBox(true).
		WithPrintSigs(true)

	colorOptions := string_format_writer.ColorOptions{OffEntirely: true}

	format := MakeBoxTransacted(
		colorOptions,
		options,
		string_format_writer.MakeBoxStringEncoder(
			string_format_writer.CliFormatTruncationNone,
			colorOptions,
		),
		ids.Abbr{},
		nil,
		nil,
		nil,
	)

	format.isArchive = true

	return format
}

// TestBlobReferenceArchiveBoxRoundTripVerifies is the regression test for the
// blob-reference inventory-list round-trip: encoding a signed
// blob-reference-carrying object through the archive box (the inventory-list
// wire form) and decoding it back MUST reproduce the signed object digest, so
// signature verification succeeds (repo-fsck / transfer / last). Runs under
// both sig purposes: v2 (digest does not cover the refs) and v3 (digest
// covers them).
func TestBlobReferenceArchiveBoxRoundTripVerifies(t1 *testing.T) {
	for _, sigPurpose := range []string{
		markl.PurposeObjectSigV2,
		markl.PurposeObjectSigV3,
	} {
		t1.Run(sigPurpose, func(t1 *testing.T) {
			t := ui.MakeT(t1)
			testBlobReferenceArchiveBoxRoundTripVerifies(&t, sigPurpose)
		})
	}
}

func testBlobReferenceArchiveBoxRoundTripVerifies(
	t *ui.T,
	sigPurpose string,
) {
	object, _ := makeSignedObjectWithBlobReferences(t, sigPurpose)

	expectedDigestPurpose := markl.GetDigestTypeForSigType(sigPurpose)

	if actual := object.GetMetadata().GetObjectDigest().GetPurposeId(); actual != expectedDigestPurpose {
		t.Fatalf(
			"expected digest purpose %q for sig purpose %q, got %q",
			expectedDigestPurpose,
			sigPurpose,
			actual,
		)
	}

	fieldsBefore := digestFields(object)
	digestBefore := object.GetMetadata().GetObjectDigest().String()

	// the exact wire form: inventory_list_store/main.go:72 builds the
	// archive box with bare print options + tai. Constructed inline (rather
	// than via MakeBoxTransactedArchive) because that constructor needs an
	// env_ui.Env solely for color options, which the archive form disables
	// anyway.
	format := makeArchiveBoxForTest()

	var buffer bytes.Buffer
	_, err := format.EncodeStringTo(object, &buffer)
	t.AssertNoError(err)

	encoded := buffer.String()
	t.Logf("wire form: %s", encoded)

	decoded, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	ringBuffer := catgut.MakeRingBuffer(strings.NewReader(encoded), 0)

	if _, err = format.ReadStringFormat(
		decoded,
		catgut.MakeRingBufferRuneScanner(ringBuffer),
	); err != nil {
		t.Fatalf("decode failed: %s", err)
	}

	fieldsAfter := digestFields(decoded)

	for key, before := range fieldsBefore {
		if after := fieldsAfter[key]; before != after {
			t.Errorf(
				"field %q diverged across round-trip:\n  before: %q\n  after:  %q",
				key,
				before,
				after,
			)
		}
	}

	// recompute the digest from the decoded metadata exactly like
	// FinalizeAndVerify does on the repo-fsck / transfer path: the digest
	// purpose derives from the decoded object's own sig purpose (the
	// deliberately mismatched fallback proves the fallback is not used)
	finalizer := object_finalizer.Make()

	if err = finalizer.FinalizeAndVerify(
		decoded,
		decoded.ObjectDigestPurposeOrDefault(markl.PurposeObjectDigestV1),
	); err != nil {
		t.Errorf(
			"FinalizeAndVerify failed after round-trip: %s\n  digest before: %s\n  digest after:  %s",
			err,
			digestBefore,
			decoded.GetMetadata().GetObjectDigest().String(),
		)
	}
}

// TestBlobReferenceMutationAfterSigning proves digest v3 covers the blob
// references: mutating a reference alias after signing MUST fail
// verification under v3, and (coexistence) MUST still pass under v2, whose
// digest does not cover the refs.
func TestBlobReferenceMutationAfterSigning(t1 *testing.T) {
	for _, testCase := range []struct {
		sigPurpose       string
		expectVerifyFail bool
	}{
		{sigPurpose: markl.PurposeObjectSigV2, expectVerifyFail: false},
		{sigPurpose: markl.PurposeObjectSigV3, expectVerifyFail: true},
	} {
		t1.Run(testCase.sigPurpose, func(t1 *testing.T) {
			t := ui.MakeT(t1)

			object, _ := makeSignedObjectWithBlobReferences(
				&t,
				testCase.sigPurpose,
			)

			var firstBlobId markl.Id

			for blobId := range object.GetMetadata().AllBlobReferences() {
				firstBlobId.ResetWith(blobId)
				break
			}

			t.AssertNoError(object.GetMetadataMutable().SetBlobReferenceAlias(
				firstBlobId,
				"tampered-alias",
			))

			finalizer := object_finalizer.Make()

			err := finalizer.FinalizeAndVerify(
				object,
				object.ObjectDigestPurposeOrDefault(markl.PurposeObjectDigestV1),
			)

			if testCase.expectVerifyFail && err == nil {
				t.Errorf(
					"expected verification failure after mutating a blob reference alias under %s",
					testCase.sigPurpose,
				)
			} else if !testCase.expectVerifyFail && err != nil {
				t.Errorf(
					"expected verification success after mutating a blob reference alias under %s, got: %s",
					testCase.sigPurpose,
					err,
				)
			}
		})
	}
}

// makeObjectWithBlobReferenceForDigest builds an unsigned object carrying a
// single aliased blob reference, holding everything except the alias fixed
// so digest comparisons isolate blob-reference coverage.
func makeObjectWithBlobReferenceForDigest(
	t *ui.T,
	tai ids.Tai,
	alias string,
) *sku.Transacted {
	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustTypeStruct("md")),
	)
	object.SetTai(tai)

	metadata := object.GetMetadataMutable()

	t.AssertNoError(markl.SetHexBytes(
		markl.FormatIdHashSha256,
		metadata.GetBlobDigestMutable(),
		[]byte(
			"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
		),
	))

	t.AssertNoError(metadata.GetTypeMutable().SetType("!toml-type-v2"))

	var blobId markl.Id

	t.AssertNoError(markl.SetHexBytes(
		markl.FormatIdHashSha256,
		&blobId,
		[]byte(
			"11500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2411",
		),
	))

	var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
	marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)
	t.AssertNoError(marshaler.Set(ids.MakeTypeString("pandoc-lua_filter")))

	metadata.AddBlobReference(blobId, typeLock)
	t.AssertNoError(metadata.SetBlobReferenceAlias(blobId, alias))

	return object
}

// TestBlobReferenceDigestCoverageV2VsV3 exercises the digest formats
// directly: two objects differing ONLY in a blob reference (its alias)
// produce identical digests under PurposeObjectDigestV2 and different
// digests under PurposeObjectDigestV3.
func TestBlobReferenceDigestCoverageV2VsV3(t1 *testing.T) {
	t := ui.MakeT(t1)

	tai := ids.NowTai()

	left := makeObjectWithBlobReferenceForDigest(&t, tai, "alias-one")
	right := makeObjectWithBlobReferenceForDigest(&t, tai, "alias-two")

	digestForPurpose := func(object *sku.Transacted, purpose string) string {
		id, repool := markl.GetId()
		defer repool()

		t.AssertNoError(object.CalculateDigestForPurpose(purpose, id))

		return id.String()
	}

	leftV2 := digestForPurpose(left, markl.PurposeObjectDigestV2)
	rightV2 := digestForPurpose(right, markl.PurposeObjectDigestV2)

	if leftV2 != rightV2 {
		t.Errorf(
			"expected identical v2 digests (refs not covered):\n  left:  %s\n  right: %s",
			leftV2,
			rightV2,
		)
	}

	leftV3 := digestForPurpose(left, markl.PurposeObjectDigestV3)
	rightV3 := digestForPurpose(right, markl.PurposeObjectDigestV3)

	if leftV3 == rightV3 {
		t.Errorf(
			"expected different v3 digests (refs covered), both: %s",
			leftV3,
		)
	}
}

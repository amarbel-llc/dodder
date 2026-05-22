//go:build test

package sku_json_fmt

import (
	"encoding/json"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

const (
	testSig1 = "ed25519_sig-n8k4wzs94yse52gnz2zw5s6jzdkqvunf8z6r4fsny0ny0kaw2t24vlkw2ns0r5aj7lzz45vxjzskth2yug7ct4a4k4c42h22hvuv7pqpkn0zj"
	testSig2 = "ed25519_sig-2a3ehc2jherahnn05tr9m62zc0sp9s8l8r7h4a9npj92rljd892a9kh62hawyujw475enup3v2z9dy0wlvam30l0lxz0j3n4huu3spsmz36y7"
	testSig3 = "ed25519_sig-anhgqrkdqnn6uzvcaj93hr7epr72v8vefv0gkrhd7ktskl6pez2cr8kwe3krrndw8lefh8a7k5dzhete4pjk72zfp4vgf8f0srpksqsy6nn8g"
	testSig4 = "ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr"

	testBlobDigest = "blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd"
)

func TestTransactedRoundTripAllLockKinds(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Build an object with every lock kind populated
	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	// Object ID
	t.AssertNoError(original.GetObjectIdMutable().Set("one/uno"))

	// Type + type lock
	t.AssertNoError(metadata.GetTypeMutable().SetType("ref-blob"))
	t.AssertNoError(metadata.GetTypeLockMutable().GetValueMutable().Set(
		testSig1,
	))

	// Tags + tag locks
	t.AssertNoError(metadata.AddTagString("project-alpha"))
	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("project-alpha"))
		tagLock := metadata.GetTagLockMutable(tag)
		t.AssertNoError(tagLock.GetValueMutable().Set(
			testSig2,
		))
	}

	// Referenced object + lock + alias
	{
		var refId ids.SeqId
		t.AssertNoError(refId.Set("two/dos"))
		t.AssertNoError(metadata.AddReference(refId))
		refLock := metadata.GetReferencedObjectLockMutable(refId)
		t.AssertNoError(refLock.GetValueMutable().Set(
			testSig3,
		))
		t.AssertNoError(metadata.SetReferenceAlias(refId, "blog-template"))
	}

	// Blob reference + type lock + alias
	{
		var blobId markl.Id
		t.AssertNoError(blobId.Set(testBlobDigest))

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		t.AssertNoError(typeLock.GetKeyMutable().SetType("img"))
		t.AssertNoError(typeLock.GetValueMutable().Set(
			testSig4,
		))

		metadata.AddBlobReference(blobId, typeLock)
		t.AssertNoError(metadata.SetBlobReferenceAlias(blobId, "hero-image"))
	}

	// Encode
	var jsonObj Transacted
	t.AssertNoError(jsonObj.FromTransacted(original, nil))

	// Verify JSON fields are populated
	encoded, err := json.MarshalIndent(jsonObj, "", "  ")
	t.AssertNoError(err)
	t.Logf("JSON:\n%s", encoded)

	// Decode into fresh object
	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	t.AssertNoError(jsonObj.ToTransacted(decoded, nil))

	decodedMeta := decoded.GetMetadataMutable()

	// Verify type lock
	t.AssertEqualStrings(
		metadata.GetTypeLock().GetValue().String(),
		decodedMeta.GetTypeLock().GetValue().String(),
	)

	// Verify tag lock
	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("project-alpha"))
		originalLock := metadata.GetTagLock(tag)
		decodedLock := decodedMeta.GetTagLock(tag)
		if originalLock == nil || decodedLock == nil {
			t.Fatal("tag lock missing after round-trip")
		}
		t.AssertEqualStrings(
			originalLock.GetValue().String(),
			decodedLock.GetValue().String(),
		)
	}

	// Verify referenced object lock + alias
	{
		var refId ids.SeqId
		t.AssertNoError(refId.Set("two/dos"))
		originalLock := metadata.GetReferencedObjectLock(refId)
		decodedLock := decodedMeta.GetReferencedObjectLock(refId)
		if originalLock == nil || decodedLock == nil {
			t.Fatal("referenced object lock missing after round-trip")
		}
		t.AssertEqualStrings(
			originalLock.GetValue().String(),
			decodedLock.GetValue().String(),
		)
		t.AssertEqualStrings(
			metadata.GetReferenceAlias(refId),
			decodedMeta.GetReferenceAlias(refId),
		)
	}

	// Verify blob reference + type lock + alias
	{
		var blobId markl.Id
		t.AssertNoError(blobId.Set(testBlobDigest))

		originalTypeLock := metadata.GetBlobReferenceTypeLock(blobId)
		decodedTypeLock := decodedMeta.GetBlobReferenceTypeLock(blobId)
		if originalTypeLock.IsEmpty() || decodedTypeLock.IsEmpty() {
			t.Fatal("blob reference type lock missing after round-trip")
		}
		t.AssertEqualStrings(
			originalTypeLock.GetKey().String(),
			decodedTypeLock.GetKey().String(),
		)
		t.AssertEqualStrings(
			originalTypeLock.GetValue().String(),
			decodedTypeLock.GetValue().String(),
		)
		t.AssertEqualStrings(
			metadata.GetBlobReferenceAlias(blobId),
			decodedMeta.GetBlobReferenceAlias(blobId),
		)
	}
}

func TestRoundTripJSONMarshalUnmarshal(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Build object with locks
	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	t.AssertNoError(original.GetObjectIdMutable().Set("one/uno"))

	t.AssertNoError(metadata.GetTypeMutable().SetType("md"))

	t.AssertNoError(metadata.GetTypeLockMutable().GetValueMutable().Set(testSig1))

	t.AssertNoError(metadata.AddTagString("project-alpha"))

	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("project-alpha"))
		tagLock := metadata.GetTagLockMutable(tag)
		t.AssertNoError(tagLock.GetValueMutable().Set(testSig2))
	}

	// Encode to struct, marshal to JSON bytes, unmarshal back, decode to object
	var jsonObj Transacted
	t.AssertNoError(jsonObj.FromTransacted(original, nil))

	bytes, err := json.Marshal(jsonObj)
	t.AssertNoError(err)

	var jsonObj2 Transacted
	t.AssertNoError(json.Unmarshal(bytes, &jsonObj2))

	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	t.AssertNoError(jsonObj2.ToTransacted(decoded, nil))

	decodedMeta := decoded.GetMetadataMutable()

	t.AssertEqualStrings(
		metadata.GetTypeLock().GetValue().String(),
		decodedMeta.GetTypeLock().GetValue().String(),
	)

	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("project-alpha"))
		decodedLock := decodedMeta.GetTagLock(tag)
		t.AssertNotNil(decodedLock, "tag lock missing after JSON marshal/unmarshal round-trip")
		t.AssertEqualStrings(testSig2, decodedLock.GetValue().String())
	}
}

func TestRoundTripPartialLocks(t1 *testing.T) {
	t := ui.MakeT(t1)

	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	t.AssertNoError(original.GetObjectIdMutable().Set("one/uno"))

	t.AssertNoError(metadata.GetTypeMutable().SetType("md"))

	// Two tags: one locked, one not
	t.AssertNoError(metadata.AddTagString("locked-tag"))
	t.AssertNoError(metadata.AddTagString("unlocked-tag"))

	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("locked-tag"))
		tagLock := metadata.GetTagLockMutable(tag)
		t.AssertNoError(tagLock.GetValueMutable().Set(testSig1))
	}

	// Reference with lock but no alias
	{
		var refId ids.SeqId
		t.AssertNoError(refId.Set("two/dos"))
		t.AssertNoError(metadata.AddReference(refId))
		refLock := metadata.GetReferencedObjectLockMutable(refId)
		t.AssertNoError(refLock.GetValueMutable().Set(testSig2))
		// No alias set
	}

	// Blob reference with type lock but no alias
	{
		var blobId markl.Id
		t.AssertNoError(blobId.Set(testBlobDigest))
		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		t.AssertNoError(typeLock.GetKeyMutable().SetType("img"))
		t.AssertNoError(typeLock.GetValueMutable().Set(testSig3))
		metadata.AddBlobReference(blobId, typeLock)
		// No alias set
	}

	// Encode → Decode
	var jsonObj Transacted
	t.AssertNoError(jsonObj.FromTransacted(original, nil))

	encoded, err := json.MarshalIndent(jsonObj, "", "  ")
	t.AssertNoError(err)
	t.Logf("JSON:\n%s", encoded)

	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	t.AssertNoError(jsonObj.ToTransacted(decoded, nil))

	decodedMeta := decoded.GetMetadataMutable()

	// Locked tag preserved
	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("locked-tag"))
		decodedLock := decodedMeta.GetTagLock(tag)
		t.AssertNotNil(decodedLock, "locked tag lock missing after round-trip")
		t.AssertEqualStrings(testSig1, decodedLock.GetValue().String())
	}

	// Unlocked tag has no lock
	{
		var tag ids.TagStruct
		t.AssertNoError(tag.Set("unlocked-tag"))
		decodedLock := decodedMeta.GetTagLock(tag)
		if decodedLock != nil && !decodedLock.GetValue().IsEmpty() {
			t.Fatalf("unlocked tag should have no lock, got: %s", decodedLock.GetValue())
		}
	}

	// Reference lock preserved, alias is empty
	{
		var refId ids.SeqId
		t.AssertNoError(refId.Set("two/dos"))
		decodedLock := decodedMeta.GetReferencedObjectLock(refId)
		t.AssertNotNil(decodedLock, "reference lock missing after round-trip")
		t.AssertEqualStrings(testSig2, decodedLock.GetValue().String())
		t.AssertEqualStrings("", decodedMeta.GetReferenceAlias(refId))
	}

	// Blob reference type lock preserved, alias is empty
	{
		var blobId markl.Id
		t.AssertNoError(blobId.Set(testBlobDigest))
		decodedTypeLock := decodedMeta.GetBlobReferenceTypeLock(blobId)
		if decodedTypeLock.IsEmpty() {
			t.Fatal("blob reference type lock missing after round-trip")
		}
		t.AssertEqualStrings("!img", decodedTypeLock.GetKey().String())
		t.AssertEqualStrings(testSig3, decodedTypeLock.GetValue().String())
		t.AssertEqualStrings("", decodedMeta.GetBlobReferenceAlias(blobId))
	}
}

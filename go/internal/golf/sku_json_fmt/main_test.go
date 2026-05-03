//go:build test

package sku_json_fmt

import (
	"encoding/json"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
)

const (
	testSig1 = "ed25519_sig-n8k4wzs94yse52gnz2zw5s6jzdkqvunf8z6r4fsny0ny0kaw2t24vlkw2ns0r5aj7lzz45vxjzskth2yug7ct4a4k4c42h22hvuv7pqpkn0zj"
	testSig2 = "ed25519_sig-2a3ehc2jherahnn05tr9m62zc0sp9s8l8r7h4a9npj92rljd892a9kh62hawyujw475enup3v2z9dy0wlvam30l0lxz0j3n4huu3spsmz36y7"
	testSig3 = "ed25519_sig-anhgqrkdqnn6uzvcaj93hr7epr72v8vefv0gkrhd7ktskl6pez2cr8kwe3krrndw8lefh8a7k5dzhete4pjk72zfp4vgf8f0srpksqsy6nn8g"
	testSig4 = "ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr"

	testBlobDigest = "blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd"
)

func TestTransactedRoundTripAllLockKinds(t1 *testing.T) {
	t := ui.T{T: t1}

	// Build an object with every lock kind populated
	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	// Object ID
	if err := original.GetObjectIdMutable().Set("one/uno"); err != nil {
		t.Fatal(err)
	}

	// Type + type lock
	if err := metadata.GetTypeMutable().SetType("ref-blob"); err != nil {
		t.Fatal(err)
	}
	if err := metadata.GetTypeLockMutable().GetValueMutable().Set(
		testSig1,
	); err != nil {
		t.Fatal(err)
	}

	// Tags + tag locks
	if err := metadata.AddTagString("project-alpha"); err != nil {
		t.Fatal(err)
	}
	{
		var tag ids.TagStruct
		if err := tag.Set("project-alpha"); err != nil {
			t.Fatal(err)
		}
		tagLock := metadata.GetTagLockMutable(tag)
		if err := tagLock.GetValueMutable().Set(
			testSig2,
		); err != nil {
			t.Fatal(err)
		}
	}

	// Referenced object + lock + alias
	{
		var refId ids.SeqId
		if err := refId.Set("two/dos"); err != nil {
			t.Fatal(err)
		}
		if err := metadata.AddReference(refId); err != nil {
			t.Fatal(err)
		}
		refLock := metadata.GetReferencedObjectLockMutable(refId)
		if err := refLock.GetValueMutable().Set(
			testSig3,
		); err != nil {
			t.Fatal(err)
		}
		if err := metadata.SetReferenceAlias(refId, "blog-template"); err != nil {
			t.Fatal(err)
		}
	}

	// Blob reference + type lock + alias
	{
		var blobId markl.Id
		if err := blobId.Set(testBlobDigest); err != nil {
			t.Fatal(err)
		}

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		if err := typeLock.GetKeyMutable().SetType("img"); err != nil {
			t.Fatal(err)
		}
		if err := typeLock.GetValueMutable().Set(
			testSig4,
		); err != nil {
			t.Fatal(err)
		}

		metadata.AddBlobReference(blobId, typeLock)
		if err := metadata.SetBlobReferenceAlias(blobId, "hero-image"); err != nil {
			t.Fatal(err)
		}
	}

	// Encode
	var jsonObj Transacted
	if err := jsonObj.FromTransacted(original, nil); err != nil {
		t.Fatal(err)
	}

	// Verify JSON fields are populated
	encoded, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("JSON:\n%s", encoded)

	// Decode into fresh object
	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	if err := jsonObj.ToTransacted(decoded, nil); err != nil {
		t.Fatal(err)
	}

	decodedMeta := decoded.GetMetadataMutable()

	// Verify type lock
	t.AssertEqualStrings(
		metadata.GetTypeLock().GetValue().String(),
		decodedMeta.GetTypeLock().GetValue().String(),
	)

	// Verify tag lock
	{
		var tag ids.TagStruct
		if err := tag.Set("project-alpha"); err != nil {
			t.Fatal(err)
		}
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
		if err := refId.Set("two/dos"); err != nil {
			t.Fatal(err)
		}
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
		if err := blobId.Set(testBlobDigest); err != nil {
			t.Fatal(err)
		}

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
	t := ui.T{T: t1}

	// Build object with locks
	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	if err := original.GetObjectIdMutable().Set("one/uno"); err != nil {
		t.Fatal(err)
	}

	if err := metadata.GetTypeMutable().SetType("md"); err != nil {
		t.Fatal(err)
	}

	if err := metadata.GetTypeLockMutable().GetValueMutable().Set(testSig1); err != nil {
		t.Fatal(err)
	}

	if err := metadata.AddTagString("project-alpha"); err != nil {
		t.Fatal(err)
	}

	{
		var tag ids.TagStruct
		if err := tag.Set("project-alpha"); err != nil {
			t.Fatal(err)
		}
		tagLock := metadata.GetTagLockMutable(tag)
		if err := tagLock.GetValueMutable().Set(testSig2); err != nil {
			t.Fatal(err)
		}
	}

	// Encode to struct, marshal to JSON bytes, unmarshal back, decode to object
	var jsonObj Transacted
	if err := jsonObj.FromTransacted(original, nil); err != nil {
		t.Fatal(err)
	}

	bytes, err := json.Marshal(jsonObj)
	if err != nil {
		t.Fatal(err)
	}

	var jsonObj2 Transacted
	if err := json.Unmarshal(bytes, &jsonObj2); err != nil {
		t.Fatal(err)
	}

	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	if err := jsonObj2.ToTransacted(decoded, nil); err != nil {
		t.Fatal(err)
	}

	decodedMeta := decoded.GetMetadataMutable()

	t.AssertEqualStrings(
		metadata.GetTypeLock().GetValue().String(),
		decodedMeta.GetTypeLock().GetValue().String(),
	)

	{
		var tag ids.TagStruct
		if err := tag.Set("project-alpha"); err != nil {
			t.Fatal(err)
		}
		decodedLock := decodedMeta.GetTagLock(tag)
		if decodedLock == nil {
			t.Fatal("tag lock missing after JSON marshal/unmarshal round-trip")
		}
		t.AssertEqualStrings(testSig2, decodedLock.GetValue().String())
	}
}

func TestRoundTripPartialLocks(t1 *testing.T) {
	t := ui.T{T: t1}

	original, repool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repool()

	metadata := original.GetMetadataMutable()

	if err := original.GetObjectIdMutable().Set("one/uno"); err != nil {
		t.Fatal(err)
	}

	if err := metadata.GetTypeMutable().SetType("md"); err != nil {
		t.Fatal(err)
	}

	// Two tags: one locked, one not
	if err := metadata.AddTagString("locked-tag"); err != nil {
		t.Fatal(err)
	}
	if err := metadata.AddTagString("unlocked-tag"); err != nil {
		t.Fatal(err)
	}

	{
		var tag ids.TagStruct
		if err := tag.Set("locked-tag"); err != nil {
			t.Fatal(err)
		}
		tagLock := metadata.GetTagLockMutable(tag)
		if err := tagLock.GetValueMutable().Set(testSig1); err != nil {
			t.Fatal(err)
		}
	}

	// Reference with lock but no alias
	{
		var refId ids.SeqId
		if err := refId.Set("two/dos"); err != nil {
			t.Fatal(err)
		}
		if err := metadata.AddReference(refId); err != nil {
			t.Fatal(err)
		}
		refLock := metadata.GetReferencedObjectLockMutable(refId)
		if err := refLock.GetValueMutable().Set(testSig2); err != nil {
			t.Fatal(err)
		}
		// No alias set
	}

	// Blob reference with type lock but no alias
	{
		var blobId markl.Id
		if err := blobId.Set(testBlobDigest); err != nil {
			t.Fatal(err)
		}
		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		if err := typeLock.GetKeyMutable().SetType("img"); err != nil {
			t.Fatal(err)
		}
		if err := typeLock.GetValueMutable().Set(testSig3); err != nil {
			t.Fatal(err)
		}
		metadata.AddBlobReference(blobId, typeLock)
		// No alias set
	}

	// Encode → Decode
	var jsonObj Transacted
	if err := jsonObj.FromTransacted(original, nil); err != nil {
		t.Fatal(err)
	}

	encoded, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("JSON:\n%s", encoded)

	decoded, repoolDecoded := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer repoolDecoded()

	if err := jsonObj.ToTransacted(decoded, nil); err != nil {
		t.Fatal(err)
	}

	decodedMeta := decoded.GetMetadataMutable()

	// Locked tag preserved
	{
		var tag ids.TagStruct
		if err := tag.Set("locked-tag"); err != nil {
			t.Fatal(err)
		}
		decodedLock := decodedMeta.GetTagLock(tag)
		if decodedLock == nil {
			t.Fatal("locked tag lock missing after round-trip")
		}
		t.AssertEqualStrings(testSig1, decodedLock.GetValue().String())
	}

	// Unlocked tag has no lock
	{
		var tag ids.TagStruct
		if err := tag.Set("unlocked-tag"); err != nil {
			t.Fatal(err)
		}
		decodedLock := decodedMeta.GetTagLock(tag)
		if decodedLock != nil && !decodedLock.GetValue().IsEmpty() {
			t.Fatalf("unlocked tag should have no lock, got: %s", decodedLock.GetValue())
		}
	}

	// Reference lock preserved, alias is empty
	{
		var refId ids.SeqId
		if err := refId.Set("two/dos"); err != nil {
			t.Fatal(err)
		}
		decodedLock := decodedMeta.GetReferencedObjectLock(refId)
		if decodedLock == nil {
			t.Fatal("reference lock missing after round-trip")
		}
		t.AssertEqualStrings(testSig2, decodedLock.GetValue().String())
		t.AssertEqualStrings("", decodedMeta.GetReferenceAlias(refId))
	}

	// Blob reference type lock preserved, alias is empty
	{
		var blobId markl.Id
		if err := blobId.Set(testBlobDigest); err != nil {
			t.Fatal(err)
		}
		decodedTypeLock := decodedMeta.GetBlobReferenceTypeLock(blobId)
		if decodedTypeLock.IsEmpty() {
			t.Fatal("blob reference type lock missing after round-trip")
		}
		t.AssertEqualStrings("!img", decodedTypeLock.GetKey().String())
		t.AssertEqualStrings(testSig3, decodedTypeLock.GetValue().String())
		t.AssertEqualStrings("", decodedMeta.GetBlobReferenceAlias(blobId))
	}
}

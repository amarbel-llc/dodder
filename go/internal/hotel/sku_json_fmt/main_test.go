//go:build test

package sku_json_fmt

import (
	"encoding/json"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
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
		"ed25519_sig-n8k4wzs94yse52gnz2zw5s6jzdkqvunf8z6r4fsny0ny0kaw2t24vlkw2ns0r5aj7lzz45vxjzskth2yug7ct4a4k4c42h22hvuv7pqpkn0zj",
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
			"ed25519_sig-2a3ehc2jherahnn05tr9m62zc0sp9s8l8r7h4a9npj92rljd892a9kh62hawyujw475enup3v2z9dy0wlvam30l0lxz0j3n4huu3spsmz36y7",
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
			"ed25519_sig-anhgqrkdqnn6uzvcaj93hr7epr72v8vefv0gkrhd7ktskl6pez2cr8kwe3krrndw8lefh8a7k5dzhete4pjk72zfp4vgf8f0srpksqsy6nn8g",
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
		if err := blobId.Set(
			"blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd",
		); err != nil {
			t.Fatal(err)
		}

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
		if err := typeLock.GetKeyMutable().SetType("img"); err != nil {
			t.Fatal(err)
		}
		if err := typeLock.GetValueMutable().Set(
			"ed25519_sig-jr7jqjh6rq0zd42n03z5vcl2grqr3eg9eqwnuwxj809h3eaxqw58mm3garf4nzenptmu9mhamhtlt9uuxsrt5wl4dshsfsnak3zvgrcelwkhr",
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
		if err := blobId.Set(
			"blake2b256-9ft3m74l5t2ppwjrvfg3wp380jqj2zfrm6zevxqx34sdethvey0s5vm9gd",
		); err != nil {
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

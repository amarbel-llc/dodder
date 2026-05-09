package local_working_copy

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
)

func TestMakeTestingProducesValidKeypair(t1 *testing.T) {
	ui.RunTestContext(t1, testMakeTestingProducesValidKeypair)
}

func testMakeTestingProducesValidKeypair(t *ui.TestContext) {
	repo := MakeTesting(t, nil)
	if repo == nil {
		t.Fatalf("MakeTesting returned nil *Repo")
	}

	pub := repo.GetImmutableConfigPublic().GetPublicKey()
	if pub.IsNull() {
		t.Fatalf("public key is null on test repo")
	}

	priv := repo.GetImmutableConfigPrivate().Blob.GetPrivateKey()
	if priv.IsNull() {
		t.Fatalf("private key is null on test repo")
	}

	hash, repoolHash := markl.FormatHashSha256.GetHash()
	defer repoolHash()
	if _, err := hash.Write([]byte("smoke")); err != nil {
		t.Fatalf("hash.Write: %v", err)
	}

	digest, repoolDigest := hash.GetMarklId()
	defer repoolDigest()

	var sig markl.Id
	if err := priv.Sign(digest, &sig, markl.PurposeRequestRepoSigV1); err != nil {
		t.Fatalf("priv.Sign: %v", err)
	}

	if err := pub.Verify(digest, sig); err != nil {
		t.Fatalf("pub.Verify: keys do not form a valid pair: %v", err)
	}
}

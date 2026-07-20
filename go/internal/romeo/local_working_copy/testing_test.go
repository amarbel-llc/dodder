package local_working_copy

import (
	"testing"

	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestMakeTestingProducesValidKeypair(t1 *testing.T) {
	ui.RunTestContext(t1, testMakeTestingProducesValidKeypair)
}

func testMakeTestingProducesValidKeypair(t *ui.TestContext) {
	repo := MakeTesting(t, nil)
	t.AssertNotNil(repo, "MakeTesting *Repo")

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
	_, err := hash.Write([]byte("smoke"))
	t.AssertNoError(err)

	digest, repoolDigest := hash.GetMarklId()
	defer repoolDigest()

	var sig markl.Id
	t.AssertNoError(priv.Sign(digest, &sig, markl.PurposeRequestRepoSigV1))

	t.AssertNoError(pub.Verify(digest, sig))
}

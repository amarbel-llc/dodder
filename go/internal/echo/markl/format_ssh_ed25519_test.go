//go:build test

package markl

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

func makeTestEd25519Signer() ed25519.PrivateKey {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	return priv
}

func TestSSHStubParseWithoutAgent(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetSSHFormatForTesting()

		var sshKey Id
		pubKey := makeTestEd25519Signer().Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		if sshKey.GetMarklFormat().GetMarklFormatId() != FormatIdEd25519SSH {
			t.Fatalf("expected format %q", FormatIdEd25519SSH)
		}
	})
}

func TestSSHStubSignReturnsConnectionError(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetSSHFormatForTesting()

		var sshKey Id
		pubKey := makeTestEd25519Signer().Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		message, repool := FormatHashSha256.GetMarklIdForString("test")
		defer repool()

		var sig Id
		err = sshKey.Sign(message, &sig, PurposeObjectSigV2)
		if err == nil {
			t.Fatal("expected error from stub Sign")
		}

		if !IsErrEd25519SSHAgentNotConnected(err) {
			t.Fatalf("expected SSH agent connection error, got: %s", err)
		}
	})
}

func TestSSHStubGetPublicKeyReturnsConnectionError(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		resetSSHFormatForTesting()

		var sshKey Id
		pubKey := makeTestEd25519Signer().Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		_, err = sshKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		if err == nil {
			t.Fatal("expected error from stub GetPublicKey")
		}

		if !IsErrEd25519SSHAgentNotConnected(err) {
			t.Fatalf("expected SSH agent connection error, got: %s", err)
		}
	})
}

func TestRegisterSSHEd25519FormatAndSign(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		priv := makeTestEd25519Signer()
		resetSSHFormatForTesting()
		RegisterSSHEd25519Format(priv)

		var sshKey Id
		pubKey := priv.Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		message, repool := FormatHashSha256.GetMarklIdForString("test message")
		defer repool()

		var sig Id
		err = sshKey.Sign(message, &sig, PurposeObjectSigV2)
		t.AssertNoError(err)
		t.AssertNoError(AssertIdIsNotNull(sig))

		var verifyPub Id
		err = verifyPub.SetPurposeId(PurposeRepoPubKeyV1)
		t.AssertNoError(err)
		err = verifyPub.SetMarklId(FormatIdEd25519Pub, pubKey)
		t.AssertNoError(err)
		err = verifyPub.Verify(message, sig)
		t.AssertNoError(err)
	})
}

func TestSSHSignThenVerify(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		priv := makeTestEd25519Signer()
		resetSSHFormatForTesting()
		RegisterSSHEd25519Format(priv)

		var sshKey Id
		pubKey := priv.Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		derivedPub, err := sshKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)

		message, repool := FormatHashSha256.GetMarklIdForString("object digest content")
		defer repool()

		var sig Id
		err = sshKey.Sign(message, &sig, PurposeObjectSigV2)
		t.AssertNoError(err)
		t.AssertNoError(AssertIdIsNotNull(sig))

		err = derivedPub.Verify(message, sig)
		t.AssertNoError(err)

		var standalonePub Id
		err = standalonePub.SetPurposeId(PurposeRepoPubKeyV1)
		t.AssertNoError(err)
		err = standalonePub.SetMarklId(FormatIdEd25519Pub, pubKey)
		t.AssertNoError(err)

		err = standalonePub.Verify(message, sig)
		t.AssertNoError(err)
	})
}

func TestSSHSignatureMatchesSoftwareSignature(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		priv := makeTestEd25519Signer()
		resetSSHFormatForTesting()
		RegisterSSHEd25519Format(priv)

		message, repool := FormatHashSha256.GetMarklIdForString("deterministic test")
		defer repool()

		var sshKey Id
		pubKey := priv.Public().(ed25519.PublicKey)
		err := sshKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = sshKey.SetMarklId(FormatIdEd25519SSH, []byte(pubKey))
		t.AssertNoError(err)

		var sshSig Id
		err = sshKey.Sign(message, &sshSig, PurposeObjectSigV2)
		t.AssertNoError(err)

		var softKey Id
		err = softKey.SetPurposeId(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)
		err = softKey.SetMarklId(FormatIdEd25519Sec, []byte(priv))
		t.AssertNoError(err)

		var softSig Id
		err = softKey.Sign(message, &softSig, PurposeObjectSigV2)
		t.AssertNoError(err)

		softPub, err := softKey.GetPublicKey(PurposeRepoPrivateKeyV1)
		t.AssertNoError(err)

		t.AssertNoError(softPub.Verify(message, sshSig))
		t.AssertNoError(softPub.Verify(message, softSig))

		t.AssertNoError(AssertEqual(sshSig, softSig))
	})
}

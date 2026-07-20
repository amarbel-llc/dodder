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
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// makeSignedObject builds a fully-signed zettel using a freshly-generated
// keypair, returning the object and the public key it was signed under. The
// signed object carries a non-null repo pubkey and object signature, so it
// exercises the `-print-sigs` provenance branch.
func makeSignedObject(
	t *ui.T,
	zettelId string,
) (*sku.Transacted, mad_domain_interfaces.MarklId) {
	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustZettelId(zettelId)),
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

	t.AssertNoError(metadata.GetTypeMutable().SetType("!md"))

	// a fake type signature so the object finalizes like a real one
	{
		typeSig := metadata.GetTypeLockMutable()
		t.AssertNoError(typeSig.GetValueMutable().GeneratePrivateKey(
			nil,
			markl.FormatIdNonceSec,
			"",
		))
	}

	config := genesis_configs.Default().Blob
	t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
		nil,
		markl.FormatIdEd25519Sec,
		markl.PurposeRepoPrivateKeyV1,
	))

	finalizer := object_finalizer.Make()
	t.AssertNoError(finalizer.FinalizeAndSignOverwrite(object, config))

	return object, config.GetPublicKey()
}

func makeProvenanceBox(options options_print.Options) *BoxTransacted {
	colorOptions := string_format_writer.ColorOptions{OffEntirely: true}

	return MakeBoxTransacted(
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
}

func encodeBox(t *ui.T, format *BoxTransacted, object *sku.Transacted) string {
	var buffer bytes.Buffer

	_, err := format.EncodeStringTo(object, &buffer)
	t.AssertNoError(err)

	return buffer.String()
}

func printSigsOptions() options_print.Options {
	return options_print.Options{}.WithPrintSigs(true)
}

// TestSelfProvenanceSelfMatchRendersHandleAtPubkey verifies that, under
// -print-sigs, an object whose repo pubkey matches the box's configured self
// pubkey renders the `<handle>@<pubkey>` self form rather than the bare
// purpose-prefixed pubkey.
func TestSelfProvenanceSelfMatchRendersHandleAtPubkey(t1 *testing.T) {
	t := ui.MakeT(t1)

	object, selfPubKey := makeSignedObject(&t, "one/uno")

	objectPubKey := object.GetMetadata().GetRepoPubKey()
	barePubKey := objectPubKey.String()
	selfForm := "myhandle@" + barePubKey
	bareForm := objectPubKey.GetPurposeId() + "@" + barePubKey

	format := makeProvenanceBox(printSigsOptions())
	format.SetSelfProvenance(selfPubKey, "myhandle")

	output := encodeBox(&t, format, object)

	if !strings.Contains(output, selfForm) {
		t.Errorf("expected self form %q in output %q", selfForm, output)
	}

	if strings.Contains(output, bareForm) {
		t.Errorf("did not expect bare form %q in self output %q", bareForm, output)
	}
}

// TestSelfProvenanceForeignRendersBarePubkey verifies that an object whose repo
// pubkey differs from the box's configured self pubkey keeps the current bare
// pubkey rendering.
func TestSelfProvenanceForeignRendersBarePubkey(t1 *testing.T) {
	t := ui.MakeT(t1)

	object, _ := makeSignedObject(&t, "one/uno")
	_, foreignSelfPubKey := makeSignedObject(&t, "two/dos")

	objectPubKey := object.GetMetadata().GetRepoPubKey()
	bareForm := objectPubKey.GetPurposeId() + "@" + objectPubKey.String()

	format := makeProvenanceBox(printSigsOptions())
	format.SetSelfProvenance(foreignSelfPubKey, "myhandle")

	output := encodeBox(&t, format, object)

	if !strings.Contains(output, bareForm) {
		t.Errorf("expected bare form %q in output %q", bareForm, output)
	}

	if strings.Contains(output, "myhandle@") {
		t.Errorf("did not expect self handle in foreign output %q", output)
	}
}

// TestSelfProvenanceUnsetRendersBarePubkey verifies that, with no self pubkey
// configured (every internal / archive box constructor), the bare pubkey is
// rendered --- the graceful degradation to today's behavior.
func TestSelfProvenanceUnsetRendersBarePubkey(t1 *testing.T) {
	t := ui.MakeT(t1)

	object, _ := makeSignedObject(&t, "one/uno")

	objectPubKey := object.GetMetadata().GetRepoPubKey()
	bareForm := objectPubKey.GetPurposeId() + "@" + objectPubKey.String()

	format := makeProvenanceBox(printSigsOptions())

	output := encodeBox(&t, format, object)

	if !strings.Contains(output, bareForm) {
		t.Errorf("expected bare form %q in output %q", bareForm, output)
	}

	if strings.Contains(output, "myhandle@") {
		t.Errorf("did not expect self handle in unset output %q", output)
	}
}

// TestSelfProvenanceLegacyObjectRendersNothing verifies that an object with no
// signature (legacy / unsigned) emits no provenance pubkey even when a self
// pubkey is configured --- the -print-sigs block is skipped entirely.
func TestSelfProvenanceLegacyObjectRendersNothing(t1 *testing.T) {
	t := ui.MakeT(t1)

	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned
	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustZettelId("one/uno")),
	)
	t.AssertNoError(object.GetMetadataMutable().GetTypeMutable().SetType("!md"))

	_, selfPubKey := makeSignedObject(&t, "two/dos")

	format := makeProvenanceBox(printSigsOptions())
	format.SetSelfProvenance(selfPubKey, "myhandle")

	output := encodeBox(&t, format, object)

	if strings.Contains(output, "ed25519_pub-") {
		t.Errorf("expected no pubkey for unsigned object, got %q", output)
	}
}

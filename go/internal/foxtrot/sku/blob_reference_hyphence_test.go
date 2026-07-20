package sku

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// blobRefStrings collects one canonical string per blob reference (alias,
// format-prefixed id, and type-lock string), so round-trip comparisons are
// byte-exact and name the diverging component.
func blobRefStrings(metadata objects.Metadata) (out []string) {
	for blobId := range metadata.AllBlobReferences() {
		out = append(
			out,
			metadata.GetBlobReferenceAlias(blobId)+
				"|"+blobId.String()+
				"|"+markl.MakeLockCoderValueNotRequired(
				metadata.GetBlobReferenceTypeLock(blobId),
			).String(),
		)
	}

	return out
}

// TestBlobReferenceHyphenceRoundTrip writes a blob-reference-carrying object
// through the hyphence formatter (writeBlobReferences) and parses it back via
// the text parser, asserting references, aliases, and type locks survive
// byte-identical. Covers: quoted alias (contains a space) with a
// value-carrying type lock, plain alias with a key-only type lock, and a
// bare reference (no alias, no lock).
func TestBlobReferenceHyphenceRoundTrip(t1 *testing.T) {
	ui.RunTestContext(t1, testBlobReferenceHyphenceRoundTrip)
}

func testBlobReferenceHyphenceRoundTrip(t *ui.TestContext) {
	envRepo := env_repo.MakeTesting(t, nil)

	object := objects.MakeBuilder().
		WithDescription("the title").
		WithType("md").
		Build()

	for _, ref := range []struct {
		blobHex       string
		typeString    string
		alias         string
		withLockValue bool
	}{
		{
			blobHex:       "11500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2411",
			typeString:    "pandoc-lua_filter",
			alias:         "filters/dodder common.lua",
			withLockValue: true,
		},
		{
			blobHex:    "22500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2422",
			typeString: "pandoc-defaults",
			alias:      "defaults/dodder-edit.yaml",
		},
		{
			blobHex: "33500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2433",
		},
	} {
		var blobId markl.Id

		t.AssertNoError(markl.SetHexBytes(
			markl.FormatIdHashSha256,
			&blobId,
			[]byte(ref.blobHex),
		))

		var typeLock markl.Lock[ids.SeqId, *ids.SeqId]

		if ref.typeString != "" {
			marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)
			t.AssertNoError(marshaler.Set(ids.MakeTypeString(ref.typeString)))

			if ref.withLockValue {
				t.AssertNoError(typeLock.GetValueMutable().GeneratePrivateKey(
					nil,
					markl.FormatIdNonceSec,
					"",
				))
			}
		}

		object.AddBlobReference(blobId, typeLock)

		if ref.alias != "" {
			t.AssertNoError(object.SetBlobReferenceAlias(blobId, ref.alias))
		}
	}

	formatFamily := makeTestTextFormatFactory(
		envRepo,
		envRepo.GetDefaultBlobStore(),
	).MakeFormatterFamily()

	encoded := writeFormat(
		t,
		&object,
		formatFamily.MetadataOnly,
		false,
		"the body",
		object_metadata_fmt_hyphence.FormatterOptions{},
		envRepo.GetDefaultBlobStore().GetDefaultHashType(),
	)

	t.Logf("hyphence form: %s", encoded)

	parsed := readFormat(
		t,
		makeTestTextFormat(envRepo, envRepo.GetDefaultBlobStore()),
		encoded,
	)

	t.AssertEqual(blobRefStrings(&object), blobRefStrings(parsed))
}

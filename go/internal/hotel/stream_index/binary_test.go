package stream_index

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestBinaryOne(t1 *testing.T) {
	t := ui.MakeT(t1)

	buffer := new(bytes.Buffer)

	coder := binaryEncoder{Sigil: ids.SigilLatest}
	decoder := makeBinary(ids.SigilLatest)
	expected, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	var expectedN int64
	var err error

	{
		t.AssertNoError(
			expected.GetObjectIdMutable().SetWithId(ids.MustZettelId("one/uno")),
		)
		expected.SetTai(ids.NowTai())
		t.AssertNoError(markl.SetHexBytes(
			markl.FormatIdHashSha256,
			expected.GetMetadataMutable().GetBlobDigestMutable(),
			[]byte(
				"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
			),
		))

		metadata := expected.GetMetadataMutable()

		t.AssertNoError(metadata.GetTypeMutable().SetType("!da-typ"))

		// generate a fake type signature
		{
			typeSig := metadata.GetTypeLockMutable()
			t.AssertNoError(typeSig.GetValueMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdNonceSec,
				"",
			))
		}

		t.AssertNoError(metadata.GetDescriptionMutable().Set("the bez"))

		t.AssertNoError(expected.AddTag(ids.MustTag("tag")))

		// TODO add mother digest field and test
		// {
		// 	id :=
		// "3c5d8b1db2149d279f4d4a6cb9457804aac6944834b62aa283beef99bccd10f0"
		// 	idReader := base64.NewDecoder(
		// 		base64.StdEncoding,
		// 		strings.NewReader(id),
		// 	)

		// 	bufferedReader, repoolBufferedReader := pool.GetBufferedReader(
		// 		idReader,
		// 	)

		// 	defer repoolBufferedReader()

		// 	t.AssertNoError(
		// 		merkle_ids.ReadFromInto(
		// 			bufferedReader,
		// 			expected.Metadata.GetMotherDigestMutable(),
		// 		),
		// 	)
		// }

		{
			config := genesis_configs.Default().Blob
			finalizer := object_finalizer.Make()

			t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdEd25519Sec,
				markl.PurposeRepoPrivateKeyV1,
			))
			t.AssertNoError(finalizer.FinalizeAndSignOverwrite(expected, config))
		}

		t.Logf("%s", expected)

		expectedN, err = coder.writeFormat(
			buffer,
			objectWithSigil{Transacted: expected},
		)
		t.AssertNoError(err)
	}

	actual := objectWithCursorAndSigil{
		objectWithSigil: objectWithSigil{
			Transacted: func() *sku.Transacted { t, _ := sku.GetTransactedPool().GetWithRepool(); return t }(), //repool:owned
		},
	}

	{
		n, err := decoder.readFormatAndMatchSigil(buffer, &actual)
		t.AssertNoError(err)
		t.Logf("%s", actual)

		t.AssertEqual(expectedN, n)
	}

	t.Logf("%s", sku.String(actual.Transacted))

	if !sku.TransactedEqualer.Equals(expected, actual.Transacted) {
		t.Errorf("expected %q but got %q", sku.String(expected), sku.String(actual.Transacted))
	}
}

func TestBinaryFieldRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	buffer := new(bytes.Buffer)

	coder := binaryEncoder{Sigil: ids.SigilLatest}
	decoder := makeBinary(ids.SigilLatest)
	expected, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	{
		t.AssertNoError(
			expected.GetObjectIdMutable().SetWithId(ids.MustZettelId("two/dos")),
		)
		expected.SetTai(ids.NowTai())
		t.AssertNoError(markl.SetHexBytes(
			markl.FormatIdHashSha256,
			expected.GetMetadataMutable().GetBlobDigestMutable(),
			[]byte(
				"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
			),
		))

		metadata := expected.GetMetadataMutable()

		t.AssertNoError(metadata.GetTypeMutable().SetType("!da-typ"))

		// generate a fake type signature
		{
			typeSig := metadata.GetTypeLockMutable()
			t.AssertNoError(typeSig.GetValueMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdNonceSec,
				"",
			))
		}

		// add a type-defined field with a non-empty TypeBlobDigest
		{
			var typeBlobDigest markl.Id
			t.AssertNoError(markl.SetHexBytes(
				markl.FormatIdHashSha256,
				&typeBlobDigest,
				[]byte(
					"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
				),
			))

			field := fields.Field{
				Key:            "status",
				Value:          "todo",
				Type:           fields.TypeUserData,
				TypeBlobDigest: typeBlobDigest,
			}

			metadata.GetIndexMutable().GetFieldsMutable().Append(field)
		}

		{
			config := genesis_configs.Default().Blob
			finalizer := object_finalizer.Make()

			t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdEd25519Sec,
				markl.PurposeRepoPrivateKeyV1,
			))
			t.AssertNoError(finalizer.FinalizeAndSignOverwrite(expected, config))
		}

		_, err := coder.writeFormat(
			buffer,
			objectWithSigil{Transacted: expected},
		)
		t.AssertNoError(err)
	}

	actual := objectWithCursorAndSigil{
		objectWithSigil: objectWithSigil{
			Transacted: func() *sku.Transacted { t, _ := sku.GetTransactedPool().GetWithRepool(); return t }(), //repool:owned
		},
	}

	{
		_, err := decoder.readFormatAndMatchSigil(buffer, &actual)
		t.AssertNoError(err)
	}

	// assert field survived round-trip
	actualFields := actual.Transacted.GetMetadataMutable().GetIndexMutable().GetFieldsMutable()

	if actualFields.Len() != 1 {
		t.Fatalf("expected 1 field but got %d", actualFields.Len())
	}

	actualField := actualFields.At(0)

	t.AssertEqualStrings("status", actualField.Key)

	t.AssertEqualStrings("todo", actualField.Value)

	t.AssertEqual(fields.TypeUserData, actualField.Type)
}

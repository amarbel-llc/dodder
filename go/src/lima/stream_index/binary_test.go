package stream_index

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/hotel/genesis_configs"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/object_finalizer"
)

func TestBinaryOne(t1 *testing.T) {
	t := ui.T{T: t1}

	buffer := new(bytes.Buffer)

	coder := binaryEncoder{Sigil: ids.SigilLatest}
	decoder := makeBinary(ids.SigilLatest)
	expected, _ := sku.GetTransactedPool().GetWithRepool()

	var expectedN int64
	var err error

	{
		t.AssertNoError(
			expected.ObjectId.SetWithId(ids.MustZettelId("one/uno")),
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
			Transacted: func() *sku.Transacted { t, _ := sku.GetTransactedPool().GetWithRepool(); return t }(),
		},
	}

	{
		n, err := decoder.readFormatAndMatchSigil(buffer, &actual)
		t.AssertNoError(err)
		t.Logf("%s", actual)

		{
			if n != expectedN {
				t.Errorf("expected %d but got %d", expectedN, n)
			}
		}
	}

	t.Logf("%s", sku.String(actual.Transacted))

	if !sku.TransactedEqualer.Equals(expected, actual.Transacted) {
		t.Errorf("expected %q but got %q", sku.String(expected), sku.String(actual.Transacted))
	}
}

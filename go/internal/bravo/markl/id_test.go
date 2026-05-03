package markl

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/blech32"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func StringHRPCombined(id domain_interfaces.MarklId) string {
	format := id.GetMarklFormat()
	data := id.GetBytes()

	if format == nil && len(data) == 0 {
		return ""
	}

	if len(data) == 0 {
		return ""
	} else {
		formatId := format.GetMarklFormatId()
		combined := make([]byte, len(formatId)+len(data))
		copy(combined, []byte(formatId))
		copy(combined[len(formatId):], data)
		bites, err := blech32.EncodeDataOnly(combined)
		errors.PanicIfError(err)
		return string(bites)
	}
}

func SetBlechCombinedHRPAndData(
	id domain_interfaces.MarklIdMutable,
	value string,
) (err error) {
	var formatId string

	var formatIdAndData []byte

	if formatIdAndData, err = blech32.DecodeDataOnly([]byte(value)); err != nil {
		err = errors.Wrapf(err, "Value: %q", value)
		return err
	}

	if bytes.HasPrefix(formatIdAndData, []byte(FormatIdHashSha256)) {
		formatId = FormatIdHashSha256
	} else if bytes.HasPrefix(formatIdAndData, []byte(FormatIdHashBlake2b256)) {
		formatId = FormatIdHashBlake2b256
	} else {
		err = errors.Errorf("unsupported format: %x", formatIdAndData)
		return err
	}

	data := formatIdAndData[len(formatId):]

	if err = id.SetMarklId(formatId, data); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func TestIdNullAndEqual(t1 *testing.T) {
	ui.RunTestContext(t1, testIdNullAndEqual)
}

func testIdNullAndEqual(t *ui.TestContext) {
	for _, formatHash := range formatHashes {
		testIdNullAndEqualFor(t, formatHash)
	}
}

func testIdNullAndEqualFor(t *ui.TestContext, formatHash FormatHash) {
	{
		t.AssertError(AssertIdIsNotNull(formatHash.null))
		t.AssertNoError(AssertIdIsNull(formatHash.null))
		t.AssertNoError(
			MakeErrLength(
				formatHash.GetSize(),
				len(formatHash.null.GetBytes()),
			),
		)
	}

	var idZero Id
	hash, _ := formatHash.Get() //repool:owned

	{
		idNull, _ := hash.GetMarklId() //repool:owned

		t.AssertNoError(AssertIdIsNull(idZero))
		t.AssertNoError(AssertIdIsNull(idNull))
		t.AssertError(AssertIdIsNotNull(idZero))
		t.AssertError(AssertIdIsNotNull(idNull))
		t.AssertNoError(AssertEqual(idZero, idNull))
		t.AssertNoError(AssertEqual(idNull, idZero))
	}

	{
		idNull, _ := formatHash.GetMarklIdForString("") //repool:owned

		t.AssertNoError(AssertIdIsNull(idZero))
		t.AssertNoError(AssertIdIsNull(idNull))
		t.AssertError(AssertIdIsNotNull(idZero))
		t.AssertError(AssertIdIsNotNull(idNull))
		t.AssertNoError(AssertEqual(idZero, idNull))
		t.AssertNoError(AssertEqual(idNull, idZero))
	}

	{
		idNull, _ := formatHash.GetBlobIdForHexString( //repool:owned
			fmt.Sprintf("%x", formatHash.null.GetBytes()),
		)

		t.AssertNoError(AssertIdIsNull(idZero))
		t.AssertNoError(AssertIdIsNull(idNull))
		t.AssertError(AssertIdIsNotNull(idZero))
		t.AssertError(AssertIdIsNotNull(idNull))
		t.AssertNoError(AssertEqual(idZero, idNull))
		t.AssertNoError(AssertEqual(idNull, idZero))
	}

	{
		idNonZero, _ := formatHash.GetMarklIdForString("nonZero") //repool:owned
		t.AssertNoError(AssertIdIsNotNull(idNonZero))
		t.AssertError(AssertIdIsNull(idNonZero))
		t.AssertError(AssertEqual(idNonZero, idZero))
		t.AssertError(AssertEqual(idZero, idNonZero))
	}
}

// TestResetWithEmptySrcClearsDstData covers a regression in (*Id).ResetWith
// that surfaced via dodder #158. When `src` is empty (no data, no format),
// the prior implementation copied src.format (nil) onto dst but left dst.data
// at its previous length because setData(empty) early-returns. A previously-
// populated dst (data=64, format=ed25519_sig) ended up as
// (data=64, format=nil), violating ADR-0001.
func TestResetWithEmptySrcClearsDstData(t1 *testing.T) {
	ui.RunTestContext(t1, testResetWithEmptySrcClearsDstData)
}

func testResetWithEmptySrcClearsDstData(t *ui.TestContext) {
	formatHash := FormatHashBlake2b256
	hash, hashRepool := formatHash.Get()
	defer hashRepool()

	if _, err := io.WriteString(hash, "regression seed"); err != nil {
		t.AssertNoError(err)
	}

	populated, populatedRepool := hash.GetMarklId()
	defer populatedRepool()

	var dst Id
	dst.ResetWith(*populated.(*Id))
	t.AssertFalse(dst.IsEmpty(), "dst should be populated after copying from non-empty src")

	var emptySrc Id
	dst.ResetWith(emptySrc)

	t.AssertTrue(dst.IsEmpty(), "dst should be empty after copying from empty src")
	t.AssertEqual(0, len(dst.GetBytes()))
	t.AssertEqual(domain_interfaces.MarklFormat(nil), dst.GetMarklFormat())
	t.AssertEqual("", dst.GetPurposeId())
}

func TestIdEncodeDecode(t1 *testing.T) {
	ui.RunTestContext(t1, testIdEncodeDecode)
}

func testIdEncodeDecode(t *ui.TestContext) {
	for _, formatHash := range formatHashes {
		hash, hashRepool := formatHash.Get()
		defer hashRepool()

		if _, err := io.WriteString(hash, "test encode decode"); err != nil {
			t.AssertNoError(err)
		}

		{
			id, repool := hash.GetMarklId()
			defer repool()

			stringValue := StringHRPCombined(id)

			t.Log(stringValue)

			t.AssertNoError(
				SetBlechCombinedHRPAndData(id, stringValue),
			)
		}
	}
}

func FuzzIdStringLen(f *testing.F) {
	f.Add("testing")
	f.Add("holidays")

	formatHash := FormatHashBlake2b256

	f.Fuzz(
		func(t1 *testing.T, input string) {
			id, repool := formatHash.GetMarklIdForString(input)
			defer repool()

			if input == "" {
				return
			}

			actual := len(id.String()) - len(formatHash.GetMarklFormatId())
			expected := 59

			if actual != expected {
				t1.Errorf("expected %d but got %d", expected, actual)
			}
		},
	)
}

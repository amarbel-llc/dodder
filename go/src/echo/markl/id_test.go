package markl

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blech32"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
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
	hash, _ := formatHash.Get()

	{
		idNull, _ := hash.GetMarklId()

		t.AssertNoError(AssertIdIsNull(idZero))
		t.AssertNoError(AssertIdIsNull(idNull))
		t.AssertError(AssertIdIsNotNull(idZero))
		t.AssertError(AssertIdIsNotNull(idNull))
		t.AssertNoError(AssertEqual(idZero, idNull))
		t.AssertNoError(AssertEqual(idNull, idZero))
	}

	{
		idNull, _ := formatHash.GetMarklIdForString("")

		t.AssertNoError(AssertIdIsNull(idZero))
		t.AssertNoError(AssertIdIsNull(idNull))
		t.AssertError(AssertIdIsNotNull(idZero))
		t.AssertError(AssertIdIsNotNull(idNull))
		t.AssertNoError(AssertEqual(idZero, idNull))
		t.AssertNoError(AssertEqual(idNull, idZero))
	}

	{
		idNull, _ := formatHash.GetBlobIdForHexString(
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
		idNonZero, _ := formatHash.GetMarklIdForString("nonZero")
		t.AssertNoError(AssertIdIsNotNull(idNonZero))
		t.AssertError(AssertIdIsNull(idNonZero))
		t.AssertError(AssertEqual(idNonZero, idZero))
		t.AssertError(AssertEqual(idZero, idNonZero))
	}
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

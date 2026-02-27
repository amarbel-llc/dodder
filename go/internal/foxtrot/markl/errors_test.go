package markl

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestErrIsNullPurposeExtractable(t1 *testing.T) {
	ui.RunTestContext(t1, testErrIsNullPurposeExtractable)
}

func testErrIsNullPurposeExtractable(t *ui.TestContext) {
	var idZero Id
	purpose := "test-purpose"

	err := AssertIdIsNotNullWithPurpose(idZero, purpose)
	t.AssertError(err)

	var errNull ErrIsNull

	if !errors.As(err, &errNull) {
		t.Fatalf("expected errors.As to extract ErrIsNull, but it did not")
	}

	if errNull.Purpose != purpose {
		t.Fatalf(
			"expected Purpose %q but got %q",
			purpose,
			errNull.Purpose,
		)
	}
}

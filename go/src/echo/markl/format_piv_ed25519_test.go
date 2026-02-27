//go:build test

package markl

import (
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func TestPIVEd25519FormatIdRegistered(t1 *testing.T) {
	ui.RunTestContext(t1, func(t *ui.TestContext) {
		purpose := GetPurpose(PurposeRepoPrivateKeyV1)
		if _, ok := purpose.formatIds[FormatIdEd25519PIV]; !ok {
			t.Fatalf(
				"format %q not registered for purpose %q",
				FormatIdEd25519PIV,
				PurposeRepoPrivateKeyV1,
			)
		}
	})
}

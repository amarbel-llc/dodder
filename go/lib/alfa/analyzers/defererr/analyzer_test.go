package defererr_test

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/analyzers/defererr"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, defererr.Analyzer, "a")
}

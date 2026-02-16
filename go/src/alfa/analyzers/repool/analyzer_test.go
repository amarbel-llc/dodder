package repool_test

import (
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/analyzers/repool"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, repool.Analyzer, "a")
}

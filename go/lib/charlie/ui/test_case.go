package ui

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/test_ui"
)

type TestCaseInfo = test_ui.TestCaseInfo

type TestCase[BLOB any] = test_ui.TestCase[BLOB]

var MakeTestCaseInfo = test_ui.MakeTestCaseInfo

//go:noinline
func MakeTestCase[BLOB any](name string, blob BLOB) TestCase[BLOB] {
	return test_ui.MakeTestCaseCallerSkip[BLOB](name, blob, 1)
}

//go:build test

package ui

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// TODO make this private and switch users over to MakeTestContext
// and add a printer

type T struct {
	*testing.T
	skip int
}

//go:noinline
func (test *T) SkipTest(args ...any) {
	if len(args) > 0 {
		test.ui(1, args...)
	}

	test.SkipNow()
}

func (test *T) Skip(skip int) *T {
	return &T{
		T:    test.T,
		skip: test.skip + skip,
	}
}

func (test *T) Run(testCaseInfo TestCaseInfo, funk func(*T)) {
	description := getTestCaseDescription(testCaseInfo)

	test.T.Run(
		description,
		func(t1 *testing.T) {
			printTestCaseInfo(testCaseInfo, description)
			funk(&T{T: t1})
		},
	)
}

// func RunTestCases[BLOB any](t *T, testCases []TestCase[BLOB]) {
// 	for _, testCase := range testCases {
// 		t.Run(
// 			testCase,
// 			func(t *T) {
// 				testCase.GetBlob().Test(t)
// 			},
// 		)
// 	}
// }

//   ___ ___
//  |_ _/ _ \
//   | | | | |
//   | | |_| |
//  |___\___/
//

//go:noinline
func (test *T) ui(skip int, args ...any) {
	Err().Caller(test.skip + 1 + skip).Print(args...)
}

//go:noinline
func (test *T) logf(skip int, format string, args ...any) {
	Err().Caller(test.skip+1+skip).Printf(format, args...)
}

//go:noinline
func (test *T) errorf(skip int, format string, args ...any) {
	test.logf(skip+1, format, args...)
	test.Fail()
}

//go:noinline
func (test *T) fatalf(skip int, format string, args ...any) {
	test.logf(skip+1, format, args...)
	test.FailNow()
}

//go:noinline
func (test *T) Log(args ...any) {
	test.ui(1, args...)
}

//go:noinline
func (test *T) Logf(format string, args ...any) {
	test.logf(1, format, args...)
}

//go:noinline
func (test *T) Errorf(format string, args ...any) {
	test.Helper()
	test.errorf(1, format, args...)
}

//go:noinline
func (test *T) Fatalf(format string, args ...any) {
	test.Helper()
	test.fatalf(1, format, args...)
}

//      _                      _
//     / \   ___ ___  ___ _ __| |_ ___
//    / _ \ / __/ __|/ _ \ '__| __/ __|
//   / ___ \\__ \__ \  __/ |  | |_\__ \
//  /_/   \_\___/___/\___|_|   \__|___/
//

// TODO-P3 move to AssertNotEqual
//
//go:noinline
func (test *T) PrintDiff(a, b any) {
	test.errorf(1, "%s", cmp.Diff(a, b, cmpopts.IgnoreUnexported(a)))
}

func PrintDiffString(test *T, a, b string) {
	test.errorf(1, "%s", cmp.Diff(a, b))
}

func TestPrintDiff[ELEMENT any](test *T, a, b ELEMENT) {
	test.errorf(1, "%s", cmp.Diff(a, b, cmpopts.IgnoreUnexported(a)))
}

//go:noinline
func (test *T) AssertEqual(a, b any, o ...cmp.Option) {
	diff := cmp.Diff(a, b, o...)

	if diff == "" {
		return
	}

	test.errorf(1, "%s", diff)
}

//go:noinline
func (test *T) AssertEqualStrings(expected, actual string) {
	test.Helper()

	if expected == actual {
		return
	}

	format := "string equality failed\n=== expected ===\n%s\n=== actual ===\n%s"
	test.errorf(1, format, expected, actual)
}

//go:noinline
func (test *T) AssertPanic(funk func()) {
	test.Helper()

	defer func() {
		if r := recover(); r == nil {
			test.errorf(2, "expected panic")
		}
	}()

	funk()
}

//go:noinline
func (test *T) AssertNoError(err error) {
	test.Helper()

	if err != nil {
		var sb strings.Builder
		CLIErrorTreeEncoder.EncodeTo(err, &sb)
		test.fatalf(1, "expected no error but got:\n%s", &sb)
	}
}

//go:noinline
func (test *T) AssertEOF(err error) {
	test.Helper()

	if err != io.EOF {
		test.fatalf(1, "expected EOF but got %q", err)
	}
}

//go:noinline
func (test *T) AssertErrorEquals(expected, actual error) {
	test.Helper()

	if actual == nil {
		test.fatalf(1, "expected %q error but got none", expected)
	}

	if !errors.Is(actual, expected) {
		test.fatalf(1, "expected %q error but got %q", expected, actual)
	}
}

//go:noinline
func (test *T) AssertError(err error) {
	test.Helper()

	if err == nil {
		test.fatalf(1, "expected an error but got none")
	}
}

//go:build test

package test_ui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

// TODO make this private and switch users over to MakeTestContext
// and add a printer

type T struct {
	*testing.T
	skip int

	// Optional. When set, output includes caller-decorated info.
	// When nil, falls back to testing.T.Log.
	Printer interfaces.Printer

	// Optional. When set, AssertNoError formats errors as a tree.
	// When nil, falls back to err.Error().
	ErrorEncoder interfaces.EncoderToWriter[error]
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
		T:            test.T,
		skip:         test.skip + skip,
		Printer:      test.Printer,
		ErrorEncoder: test.ErrorEncoder,
	}
}

func (test *T) Run(testCaseInfo TestCaseInfo, funk func(*T)) {
	description := GetTestCaseDescription(testCaseInfo)

	test.T.Run(
		description,
		func(t1 *testing.T) {
			PrintTestCaseInfo(testCaseInfo, description)
			funk(&T{
				T:            t1,
				Printer:      test.Printer,
				ErrorEncoder: test.ErrorEncoder,
			})
		},
	)
}

//   ___ ___
//  |_ _/ _ \
//   | | | | |
//   | | |_| |
//  |___\___/
//

//go:noinline
func (test *T) ui(skip int, args ...any) {
	if test.Printer != nil {
		test.Printer.Caller(test.skip + 1 + skip).Print(args...)
		return
	}

	test.Helper()
	test.T.Log(args...)
}

//go:noinline
func (test *T) logf(skip int, format string, args ...any) {
	if test.Printer != nil {
		test.Printer.Caller(test.skip+1+skip).Printf(format, args...)
		return
	}

	test.Helper()
	test.T.Logf(format, args...)
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
		var msg string

		if test.ErrorEncoder != nil {
			var sb strings.Builder
			test.ErrorEncoder.EncodeTo(err, &sb)
			msg = sb.String()
		} else {
			msg = fmt.Sprintf("%s", err)
		}

		test.fatalf(1, "expected no error but got:\n%s", msg)
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

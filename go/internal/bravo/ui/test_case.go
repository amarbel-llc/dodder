package ui

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/stack_frame"
)

type TestCaseInfo interface {
	GetName() string
	GetStackFrame() stack_frame.Frame
}

type TestCase[BLOB any] struct {
	testCaseInfo
	blob BLOB
}

//go:noinline
func MakeTestCase[BLOB any](name string, blob BLOB) TestCase[BLOB] {
	return TestCase[BLOB]{
		testCaseInfo: testCaseInfo{
			name:       name,
			stackFrame: stack_frame.MustFrame(1),
		},
		blob: blob,
	}
}

var _ TestCaseInfo = TestCase[string]{}

func (testCase TestCase[BLOB]) GetBlob() BLOB {
	return testCase.blob
}

type testCaseInfo struct {
	name       string
	stackFrame stack_frame.Frame
}

var _ TestCaseInfo = testCaseInfo{}

//go:noinline
func MakeTestCaseInfo(name string) testCaseInfo {
	return testCaseInfo{
		name:       name,
		stackFrame: stack_frame.MustFrame(1),
	}
}

func (testCase testCaseInfo) GetName() string {
	return testCase.name
}

func (testCase testCaseInfo) GetStackFrame() stack_frame.Frame {
	return testCase.stackFrame
}

func getTestCaseDescription(testCaseInfo TestCaseInfo) string {
	description := testCaseInfo.GetName()

	if description == "" {
		description = fmt.Sprintf("%v", testCaseInfo)
	}

	return description
}

func printTestCaseInfo(testCaseInfo TestCaseInfo, description string) {
	fmt.Fprintf(
		os.Stderr,
		"%s running test case %q\n",
		testCaseInfo.GetStackFrame().StringNoFunctionName(),
		description,
	)
}

//go:build test

package ui

import (
	"os"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
	"code.linenisgreat.com/dodder/go/lib/alfa/test_ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type t = T

type TestContext struct {
	t

	Context errors.Context
}

func RunTestContext(
	t *testing.T,
	run func(*TestContext),
) {
	testContext := makeTestContext(t, errors.MakeContextDefault())

	if err := testContext.Context.Run(
		func(_ errors.Context) {
			run(testContext)
		},
	); err != nil {
		_, frames := testContext.Context.CauseWithStackFrames()
		err = stack_frame.MakeErrorTreeOrErr(err, frames...)
		CLIErrorTreeEncoder.EncodeTo(err, os.Stderr)
		testContext.Skip(1).Fatalf("test context failed: %s", err)
	}
}

func makeTestContext(
	t *testing.T,
	ctx errors.Context,
) *TestContext {
	testContext := &TestContext{
		t: T{
			T:            t,
			Printer:      Err(),
			ErrorEncoder: CLIErrorTreeEncoder,
		},
		Context: ctx,
	}

	return testContext
}

func (testContext *TestContext) Skip(skip int) *TestContext {
	return &TestContext{
		t:       *testContext.t.Skip(skip),
		Context: testContext.Context,
	}
}

func (testContext *TestContext) Run(
	testCaseInfo TestCaseInfo,
	funk func(*TestContext),
) {
	description := test_ui.GetTestCaseDescription(testCaseInfo)

	testContext.T.Run(
		description,
		func(t1 *testing.T) {
			test_ui.PrintTestCaseInfo(testCaseInfo, description)
			childContext := errors.MakeContext(testContext.Context)
			childTestContext := makeTestContext(t1, childContext)
			funk(childTestContext)
		},
	)
}

package ui

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type testCaseCLITreeState struct {
	TestCaseInfo
	input    error
	expected string
}

func TestCLITreeForwards(t *testing.T) {
	RunTestContext(t, testCLITreeForwards)
}

func testCLITreeForwards(t *TestContext) {
	type testCase = testCaseCLITreeState

	testCases := []testCase{
		{
			TestCaseInfo: MakeTestCaseInfo("error group three"),
			input: errors.Group{
				newPkgError("one"),
				newPkgError("two"),
				newPkgError("three"),
			},
			expected: `error group: 3 errors
├── one
├── two
└── three
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"error group three with nested child",
			),
			input: errors.Group{
				newPkgError("one"),
				newPkgError("two"),
				errors.Group{
					newPkgError("three"),
				},
			},
			expected: `error group: 3 errors
├── one
├── two
└── three
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"error group three with double nested child",
			),
			input: errors.Group{
				newPkgError("one"),
				newPkgError("two"),
				errors.Group{
					errors.Err501NotImplemented.WrapIncludingHTTP(
						newPkgError("inner"),
					),
				},
			},
			expected: `error group: 3 errors
├── one
├── two
└── errors.HTTP: 501 Not Implemented
    └── inner
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"error group with one child",
			),
			input: errors.Group{
				newPkgError("one"),
			},
			expected: "one\n",
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"error no stack",
			),
			input: errors.WithoutStack(
				errors.Wrap(newPkgError("one")),
			),
			expected: "one\n",
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"nested group followed by sibling",
			),
			input: errors.Group{
				errors.Group{
					newPkgError("a"),
					newPkgError("b"),
				},
				newPkgError("c"),
			},
			expected: `error group: 2 errors
├── error group: 2 errors
│   ├── a
│   └── b
└── c
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"nested group as first child with trailing siblings",
			),
			input: errors.Group{
				errors.Group{
					newPkgError("a"),
					newPkgError("b"),
				},
				newPkgError("c"),
				newPkgError("d"),
			},
			expected: `error group: 3 errors
├── error group: 2 errors
│   ├── a
│   └── b
├── c
└── d
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"wrapper in middle of group",
			),
			input: errors.Group{
				errors.Err501NotImplemented.WrapIncludingHTTP(
					newPkgError("inner"),
				),
				newPkgError("two"),
				newPkgError("three"),
			},
			expected: `error group: 3 errors
├── errors.HTTP: 501 Not Implemented
│   └── inner
├── two
└── three
`,
		},
		{
			TestCaseInfo: MakeTestCaseInfo(
				"deeply nested single-child groups collapse",
			),
			input: errors.Group{
				errors.Group{
					errors.Group{
						newPkgError("only"),
					},
				},
			},
			expected: "only\n",
		},
		// TODO figure out how to include stack info stabley
		// {
		// 	TestCaseInfo: MakeTestCaseInfo(
		// 		"one error with stack",
		// 	),
		// 	input: errors.Wrap(newPkgError("one")),
		// 	expected: `one
		// └── # TestCLITreeForwards
		// │     src/charlie/error_coders/cli_tree_state_test.go:94
		// `,
		// },
		// {
		// 	TestCaseInfo: MakeTestCaseInfo(
		// 		"one in group with stack",
		// 	),
		// 	input: errors.Wrap(errors.Group{newPkgError("one")}),
		// 	expected: `one
		// └── # TestCLITreeForwards
		// │     src/charlie/error_coders/cli_tree_state_test.go:104
		// `,
		// },
		// {
		// 	TestCaseInfo: MakeTestCaseInfo(
		// 		"one with stack in group with stack",
		// 	),
		// 	input: errors.Wrap(errors.Group{errors.Errorf("one")}),
		// 	expected: `one
		// └── # TestCLITreeForwards
		// │     src/charlie/error_coders/cli_tree_state_test.go:114
		// `,
		// },
	}

	for _, testCase := range testCases {
		t.Run(
			testCase,
			func(t *TestContext) {
				var stringBuilder strings.Builder

				bufferedWriter, repool := pool.GetBufferedWriter(&stringBuilder)
				defer repool()

				coder := cliTreeState{
					bufferedWriter: bufferedWriter,
				}

				{
					err := coder.encode(testCase.input)

					if coder.bytesWritten == 0 {
						t.Errorf("expected non-zero bytes written")
					}

					t.AssertNoError(err)
				}

				actual := stringBuilder.String()

				t.AssertEqualStrings(testCase.expected, actual)
			},
		)
	}
}

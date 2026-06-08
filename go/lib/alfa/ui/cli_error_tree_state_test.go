package ui

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	dewey_ui "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

type testCaseCLITreeState struct {
	dewey_ui.TestCaseInfo
	input    error
	expected string
}

func TestCLITreeForwards(t *testing.T) {
	dewey_ui.RunTestContext(t, testCLITreeForwards)
}

func testCLITreeForwards(t *dewey_ui.TestContext) {
	type testCase = testCaseCLITreeState

	testCases := []testCase{
		{
			TestCaseInfo: dewey_ui.MakeTestCaseInfo("error group three"),
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
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
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
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
				"error group three with double nested child",
			),
			input: errors.Group{
				newPkgError("one"),
				newPkgError("two"),
				errors.Group{
					errors.Err501NotImplemented.Wrap(
						newPkgError("inner"),
					).HTTPRender(),
				},
			},
			expected: `error group: 3 errors
├── one
├── two
└── HTTP: 501 Not Implemented
    └── inner
`,
		},
		{
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
				"error group with one child",
			),
			input: errors.Group{
				newPkgError("one"),
			},
			expected: "one\n",
		},
		{
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
				"error no stack",
			),
			input: errors.WithoutStack(
				errors.Wrap(newPkgError("one")),
			),
			expected: "one\n",
		},
		{
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
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
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
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
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
				"wrapper in middle of group",
			),
			input: errors.Group{
				errors.Err501NotImplemented.Wrap(
					newPkgError("inner"),
				).HTTPRender(),
				newPkgError("two"),
				newPkgError("three"),
			},
			expected: `error group: 3 errors
├── HTTP: 501 Not Implemented
│   └── inner
├── two
└── three
`,
		},
		{
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
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
		{
			// dewey HTTPStatusError post-#107 returns its underlying
			// message verbatim from Error(), so an HTTP-tagged leaf like
			// ErrNeedsMerge (Err409Conflict.Errorf("...")) is a
			// transparent single-child wrapper. The encoder must collapse
			// it instead of rendering the same message twice (once as the
			// node, once as its sole child) — the #351 regression.
			TestCaseInfo: dewey_ui.MakeTestCaseInfo(
				"transparent http wrapper collapses to one line",
			),
			input:    errors.Err409Conflict.Errorf("import failed with conflicts, merging required"),
			expected: "import failed with conflicts, merging required\n",
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
			func(t *dewey_ui.TestContext) {
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

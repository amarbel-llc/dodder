package queries

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
)

func TestQuery(t1 *testing.T) {
	type testCase struct {
		ui.TestCaseInfo
		description, expected, expectedOptimized string
		defaultGenre                             ids.Genre
		inputs                                   []string
	}

	t := ui.T{T: t1}

	testCases := []testCase{
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[[test,house] home]",
			inputs:       []string{"[test, house] home"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[[test,house] home wow]",
			inputs:       []string{"[test, house] home", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[^[test,house] home wow]",
			inputs:       []string{"^[test, house] home", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[[test,house] ^home wow]",
			inputs:       []string{"[test, house] ^home", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[[test,^house] home wow]",
			inputs:       []string{"[test, ^house] home", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[[test,house] home ^wow]",
			inputs:       []string{"[test, house] home", "^wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[^[[test,house] home] wow]",
			inputs:       []string{"^[[test, house] home]", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "^[[test,house] home]:Zettel wow",
			inputs:       []string{"^[[test, house] home]:z", "wow"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "[!md,home]:Zettel",
			inputs:       []string{"[!md,home]:z"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "!md?Zettel",
			inputs:       []string{"!md?z"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "ducks:Tag [!md house]+?Zettel",
			inputs:       []string{"!md?z", "house+z", "ducks:e"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "ducks:Tag [!md house]?Zettel",
			inputs:       []string{"ducks:Tag [!md house]?Zettel"},
		},
		{
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			expected:     "ducks:Tag [=!md house]?Zettel",
			inputs:       []string{"ducks:Tag [=!md house]?Zettel"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "ducks:Tag [=!md house wow]:?Zettel",
			expected:          "ducks:Tag [=!md house wow]:?Zettel",
			inputs: []string{
				"ducks:Tag [=!md house]?Zettel wow:Zettel",
			},
		},
		{ // TODO try to make this expect `one/uno.zettel`
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "one/uno:.Zettel",
			expected:          "one/uno:.Zettel",
			inputs:            []string{"one/uno.zettel"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "one/uno:Zettel",
			expected:          "one/uno:Zettel",
			defaultGenre:      ids.MakeGenre(genres.Zettel),
			inputs:            []string{"one/uno"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "one/uno:Zettel",
			expected:          "one/uno:Zettel",
			inputs:            []string{"one/uno:z"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: ":Config",
			expected:          ":Config",
			inputs:            []string{":konfig"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: ":Zettel",
			expected:          ":Zettel",
			inputs:            []string{":z"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: ":Repo",
			expected:          ":Repo",
			inputs:            []string{":k"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "one/uno:+Zettel",
			expected:          "one/uno:+Zettel",
			inputs:            []string{"one/uno+"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: "[one/dos, one/uno]:Zettel",
			expected:          "[one/dos, one/uno]:Zettel",
			inputs:            []string{"one/uno", "one/dos"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			expectedOptimized: ":Type :Tag :Zettel",
			expected:          ":Type,Tag,Zettel",
			inputs:            []string{":z,t,e"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: ":Blob :Type :Tag :Zettel :Config :InventoryList :Repo",
			expected:          ":Blob,Type,Tag,Zettel,Config,InventoryList,Repo",
			inputs:            []string{},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: ":Blob :Type :Tag :Zettel :Config :InventoryList :Repo",
			expected:          ":Blob,Type,Tag,Zettel,Config,InventoryList,Repo",
			inputs:            []string{":"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: "2109504781.792086:InventoryList",
			expected:          "2109504781.792086:InventoryList",
			inputs:            []string{"[2109504781.792086]:b"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: "^etikett-two.Zettel",
			expected:          "^etikett-two.Zettel",
			inputs:            []string{"^etikett-two.z"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: "!md.Blob !md.Type !md.Tag !md.Zettel !md.Config !md.InventoryList !md.Repo",
			expected:          "!md.Blob,Type,Tag,Zettel,Config,InventoryList,Repo",
			inputs:            []string{"!md."},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: "-etikett-two.Zettel",
			expected:          "-etikett-two.Zettel",
			inputs:            []string{"-etikett-two.z"},
		},
		{
			TestCaseInfo:      ui.MakeTestCaseInfo(""),
			defaultGenre:      ids.MakeGenre(genres.All()...),
			expectedOptimized: "/repo:Repo",
			expected:          "/repo:Repo",
			inputs:            []string{"/repo:k"},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase,
			func(t *ui.T) {
				sut := (&Builder{}).WithOptions(
					BuilderOptionDefaultGenres(testCase.defaultGenre.Slice()...),
				)

				m, err := sut.BuildQueryGroup(testCase.inputs...)

				t.AssertNoError(err)
				actual := m.String()

				if testCase.expected != actual {
					t.Log("expected")
					t.AssertEqual(testCase.expected, actual)
				}

				if testCase.expectedOptimized == "" {
					return
				}

				actualOptimized := m.StringOptimized()

				if testCase.expectedOptimized != actualOptimized {
					t.Log(m.StringDebug())
					t.Log("expectedOptimized")
					t.AssertEqual(testCase.expectedOptimized, actualOptimized)
				}
			},
		)
	}
}

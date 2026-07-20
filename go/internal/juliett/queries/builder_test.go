package queries

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestFieldQuery(t1 *testing.T) {
	t := ui.MakeT(t1)

	t.Run(
		ui.MakeTestCaseInfo("field query string"),
		func(t *ui.T) {
			sut := (&Builder{}).WithOptions(
				BuilderOptionDefaultGenres(genres.Zettel),
			)

			m, err := sut.BuildQueryGroup("status=completed:z")
			t.AssertNoError(err)

			actual := m.String()
			t.AssertEqual("status=completed:Zettel", actual)
		},
	)

	t.Run(
		ui.MakeTestCaseInfo("negated field query string"),
		func(t *ui.T) {
			sut := (&Builder{}).WithOptions(
				BuilderOptionDefaultGenres(genres.Zettel),
			)

			m, err := sut.BuildQueryGroup("^status=completed:z")
			t.AssertNoError(err)

			actual := m.String()
			t.AssertEqual("^status=completed:Zettel", actual)
		},
	)

	t.Run(
		ui.MakeTestCaseInfo("field query matching"),
		func(t *ui.T) {
			sut := (&Builder{}).WithOptions(
				BuilderOptionDefaultGenres(genres.Zettel),
			)

			m, err := sut.BuildQueryGroup("status=completed:z")
			t.AssertNoError(err)

			object, repool := sku.GetTransactedPool().GetWithRepool()
			defer repool()

			object.ObjectId.Genre = genres.Zettel
			object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
				fields.Field{
					Key:   "status",
					Value: "completed",
				},
			)

			t.AssertTrue(m.containsSku(object), "expected query to match object with status=completed field")
		},
	)

	t.Run(
		ui.MakeTestCaseInfo("field query non-matching"),
		func(t *ui.T) {
			sut := (&Builder{}).WithOptions(
				BuilderOptionDefaultGenres(genres.Zettel),
			)

			m, err := sut.BuildQueryGroup("status=completed:z")
			t.AssertNoError(err)

			object, repool := sku.GetTransactedPool().GetWithRepool()
			defer repool()

			object.ObjectId.Genre = genres.Zettel
			object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
				fields.Field{
					Key:   "status",
					Value: "cancelled",
				},
			)

			t.AssertFalse(m.containsSku(object), "expected query not to match object with status=cancelled field")
		},
	)

	t.Run(
		ui.MakeTestCaseInfo("negated field query matching"),
		func(t *ui.T) {
			sut := (&Builder{}).WithOptions(
				BuilderOptionDefaultGenres(genres.Zettel),
			)

			m, err := sut.BuildQueryGroup("^status=cancelled:z")
			t.AssertNoError(err)

			object, repool := sku.GetTransactedPool().GetWithRepool()
			defer repool()

			object.ObjectId.Genre = genres.Zettel
			object.GetMetadataMutable().GetIndexMutable().GetFieldsMutable().Append(
				fields.Field{
					Key:   "status",
					Value: "completed",
				},
			)

			t.AssertTrue(m.containsSku(object), "expected negated query to match object without status=cancelled field")
		},
	)
}

// Note: infix negation syntax key^=value does not work because ^ is
// operatorTypeSoloSeq in doddish, breaking the token sequence. Use prefix
// syntax ^key=value instead (see #98).

func TestQuery(t1 *testing.T) {
	type testCase struct {
		ui.TestCaseInfo
		description, expected, expectedOptimized string
		defaultGenre                             ids.Genre
		inputs                                   []string
		expectErr                                error
	}

	t := ui.MakeT(t1)

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
			// config left the query surface (FDR 0020): an explicit konfig
			// genre token now errors rather than building a :Config query.
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			inputs:       []string{":konfig"},
			expectErr:    ErrConfigNotQueryable,
		},
		{
			// bare konfig object id likewise errors.
			TestCaseInfo: ui.MakeTestCaseInfo(""),
			inputs:       []string{"konfig"},
			expectErr:    ErrConfigNotQueryable,
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

				if testCase.expectErr != nil {
					if !errors.Is(err, testCase.expectErr) {
						t.Errorf(
							"expected error %q but got %q",
							testCase.expectErr,
							err,
						)
					}

					return
				}

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

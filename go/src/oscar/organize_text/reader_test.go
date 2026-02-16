package organize_text

import (
	"os"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/options_print"
	"code.linenisgreat.com/dodder/go/src/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/src/echo/descriptions"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/box_format"
)

// TODO transition this to ui.TestContext, and add a config object with a public
// key

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func makeZettelId(t *ui.T, v string) (k *ids.ObjectId) {
	var err error

	var h ids.ZettelId

	if err = h.Set(v); err != nil {
		t.Fatalf("%s", err)
	}

	return ids.MustObjectId(h)
}

func makeDescription(t *ui.T, v string) (b descriptions.Description) {
	var err error

	if err = b.Set(v); err != nil {
		t.Fatalf("%s", err)
	}

	return b
}

func makeObjWithZettelIdAndDescription(
	t *ui.T,
	zettelId string,
	description string,
) (object *obj) {
	object = &obj{
		sku: func() sku.SkuType { s, _ := sku.MakeSkuType(); return s }(),
	}

	object.sku.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().ResetWith(
		makeDescription(
			t,
			description,
		),
	)

	object.GetSkuExternal().ObjectId.SetWithId(makeZettelId(t, zettelId))

	// TODO add pubkeys to assignment reader
	// {
	// 	config := genesis_configs.Default().Blob
	// 	t.AssertNoError(config.GeneratePrivateKey())
	// 	t.AssertNoError(object.sku.GetSkuExternal().Sign(config))
	// 	t.AssertNoError(object.sku.GetSku().Sign(config))
	// }

	return object
}

func makeAssignmentLineReader() reader {
	return reader{
		options: Options{
			wasMade:       true,
			Config:        &repo_configs.DryRunOnly{},
			ObjectFactory: (&sku.ObjectFactory{}).SetDefaultsIfNecessary(),
			fmtBox: box_format.MakeBoxCheckedOut(
				string_format_writer.ColorOptions{},
				options_print.Options{},
				nil,
				ids.Abbr{},
				nil,
				nil,
				nil,
			),
		},
	}
}

func assertEqualObjects(t *ui.T, expected, actual Objects) {
	t = t.Skip(1)

	actual.Sort()
	expected.Sort()

	if len(actual) != len(expected) {
		t.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
	}

	for i := range actual {
		// actualObj, expectedObj := actual[i].External.GetSkuExternal(),
		// expected[i].External.GetSkuExternal()
		actualObj := sku.StringMetadataSansTai(actual[i].GetSkuExternal())
		expectedObj := sku.StringMetadataSansTai(expected[i].GetSkuExternal())

		if actualObj != expectedObj {
			t.Errorf("\nexpected: %#v\n  actual: %#v", expectedObj, actualObj)
		}
	}
}

func TestAssignmentLineReaderOneHeadingNoZettels(t1 *testing.T) {
	t := ui.T{T: t1}
	input := `# wow
    `

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t.Errorf("expected read amount to be greater than 0")
	}

	t.AssertNoError(err)

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("wow"))

		if len(sub.root.Children) < 1 {
			t.Fatalf("expected exactly 1 child")
		}

		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected) {
			t.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}
}

func TestAssignmentLineReader2Heading2Zettels(t1 *testing.T) {
	t := &ui.T{T: t1}

	input := `# wow
- [one/wow] uno
- [dos/wow] two/wow
`

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t1.Errorf("expected read amount to be greater than 0")
	}

	if err != nil {
		t1.Fatalf("expected no error but got %q", err)
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("wow"))
		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf(
				"\nexpected: %s\n  actual: %s",
				quiter.StringCommaSeparated(expected),
				quiter.StringCommaSeparated(actual),
			)
		}
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "dos/wow", "two/wow"))

		actual := sub.root.Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}
}

func TestAssignmentLineReader1_1Heading2_2Zettels(t1 *testing.T) {
	t := &ui.T{T: t1}

	input := `# wow
## sub-wow
- [one/wow] uno
- [dos/wow] two/wow
`

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t.Errorf("expected read amount to be greater than 0")
	}

	if err != nil {
		t.Fatalf("expected no error but got %q", err)
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("wow"))
		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected) {
			t.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("sub-wow"))

		if sub.root != sub.root.Children[0].Parent {
			t.Fatalf("%v, %v", sub.root, sub.root.Children[0].Parent)
		}

		l := len(sub.root.Children[0].Children)

		if l != 1 {
			t.Fatalf("\nexpected: %d\n  actual: %d", 1, l)
		}

		actual := sub.root.Children[0].Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected) {
			t.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(
			makeObjWithZettelIdAndDescription(t, "dos/wow", "two/wow"),
		)

		actual := sub.root.Children[0].Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}
}

func TestAssignmentLineReader2_1Heading2_2_2Zettels(t1 *testing.T) {
	t := &ui.T{T: t1}

	input := `# wow

- [one/wow] uno
- [dos/wow] two/wow

## sub-wow

- [three/wow] tres
- [four/wow] quatro

# cow

- [one/wow] uno
- [dos/wow] two/wow
`

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t1.Errorf("expected read amount to be greater than 0")
	}

	if err != nil {
		t1.Fatalf("expected no error but got %q", err)
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("wow"))
		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("sub-wow"))

		l := len(sub.root.Children[0].Children)
		if l != 1 {
			t1.Fatalf("\nexpected: %d\n  actual: %d", 1, l)
		}

		actual := sub.root.Children[0].Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected,
		) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("cow"))
		actual := sub.root.Children[1].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(
			actual,
			expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "dos/wow", "two/wow"))

		actual := sub.root.Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "dos/wow", "two/wow"))

		actual := sub.root.Children[1].Objects

		assertEqualObjects(t, expected, actual)
	}
}

func TestAssignmentLineReader2_1Heading2_2_2ZettelsOffset(t1 *testing.T) {
	t := &ui.T{T: t1}

	input := `
- [one/wow] uno
- [dos/wow] two/wow

## sub-wow

- [three/wow] tres
- [four/wow] quatro

## sub-cow

- [one/wow] uno
- [dos/wow] two/wow
`

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t1.Errorf("expected read amount to be greater than 0")
	}

	if err != nil {
		t1.Fatalf("expected no error but got %q", err)
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("sub-wow"))
		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("sub-cow"))

		l := len(sub.root.Children)
		expLen := 2
		if l != expLen {
			t1.Fatalf("\nexpected: %d\n  actual: %d", expLen, l)
		}

		actual := sub.root.Children[1].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "four/wow", "quatro"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "three/wow", "tres"))

		actual := sub.root.Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}

	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "dos/wow", "two/wow"))

		actual := sub.root.Children[1].Objects

		assertEqualObjects(t, expected, actual)
	}
}

func TestAssignmentLineReaderBigCheese(t1 *testing.T) {
	t := &ui.T{T: t1}

	input := `# task
- [one/wow] uno
- [two/wow] dos/wow
## priority-1
### w-2022-07-09
- [three/wow] tres
###
- [four/wow] quatro
## priority-2
- [five/wow] cinco
- [six/wow] seis
`

	sr := strings.NewReader(input)
	sub := makeAssignmentLineReader()

	n, err := sub.ReadFrom(sr)

	if n == 0 {
		t1.Errorf("expected read amount to be greater than 0")
	}

	if err != nil {
		t1.Fatalf("expected no error but got %q", err)
	}

	// `# task
	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("task"))
		actual := sub.root.Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	// - [one/wow] uno
	// - [two/wow] dos/wow
	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "one/wow", "uno"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "two/wow", "dos/wow"))

		actual := sub.root.Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}

	// ## priority-1
	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("priority-1"))

		e := 2
		l := len(sub.root.Children[0].Children)
		if l != e {
			t1.Fatalf("\nexpected: %d\n  actual: %d", e, l)
		}

		actual := sub.root.Children[0].Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	// ### w-2022-07-09
	{
		expected := ids.MakeTagSetFromSlice(ids.MustTag("w-2022-07-09"))
		actual := sub.root.Children[0].Children[0].Children[0].Transacted.GetMetadata().GetTags()

		if !quiter_set.Equals(actual, expected) {
			t1.Errorf("\nexpected: %s\n  actual: %s", expected, actual)
		}
	}

	// - [three/wow] tres
	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "three/wow", "tres"))

		actual := sub.root.Children[0].Children[0].Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}

	// ##
	// - [four/wow] quatro
	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "four/wow", "quatro"))

		actual := sub.root.Children[0].Children[0].Objects

		assertEqualObjects(t, expected, actual)
	}

	// ## priority-2
	// - [five/wow] cinco
	// - [six/wow] seis
	// `
	{
		expected := make(Objects, 0)
		expected.Add(makeObjWithZettelIdAndDescription(t, "five/wow", "cinco"))
		expected.Add(makeObjWithZettelIdAndDescription(t, "six/wow", "seis"))

		actual := sub.root.Children[0].Children[1].Objects

		assertEqualObjects(t, expected, actual)
	}
}

package tridex

import (
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type testStringer string

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

type t ui.T

func (t t) assertLen(sut interfaces.Tridex, d int) {
	if sut.Len() != d {
		t.Fatalf("expected count %d but got %d", d, sut.Len())
	}
}

func (t t) assertNotContains(sut interfaces.Tridex, v string) {
	if sut.ContainsAbbreviation(v) {
		t.Fatalf("expected to not contain %q", v)
	}
}

func (t t) assertContains(sut interfaces.Tridex, v string) {
	if !sut.ContainsAbbreviation(v) {
		t.Fatalf("expected to contain %q", v)
	}
}

func (t t) assertContainsExpansion(sut interfaces.Tridex, v string) {
	if !sut.ContainsExpansion(v) {
		t.Fatalf("expected to contain exactly %q", v)
	}
}

func (t t) assertNotContainsExpansion(sut interfaces.Tridex, v string) {
	if sut.ContainsExpansion(v) {
		t.Fatalf("expected not to contain exactly %q", v)
	}
}

func TestContains(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"123456",
		"654321",
		"5",
	)

	expectedContains := []string{
		"123456",
		"654321",
		"5",
	}

	for _, e := range expectedContains {
		if !sut.ContainsExpansion(e) {
			t.Errorf("expected %v to contain %s", sut, e)
		}
	}

	expectedNotContains := []string{
		"1",
		"12",
		"123",
		"1234",
		"12345",
		"1234567",
		"12X45",
		"3",
		"6",
		"65",
		"654",
		"6543",
		"65432",
	}

	for _, e := range expectedNotContains {
		if sut.ContainsExpansion(e) {
			t.Errorf("expected %v to not contain %s", sut, e)
		}
	}
}

func TestLen(t1 *testing.T) {
	t := t(ui.T{T: t1})

	sut := Make("one")
	t.assertLen(sut, 1)
	t.assertContains(sut, "o")
	t.assertContains(sut, "on")
	t.assertContains(sut, "one")
	t.assertContainsExpansion(sut, "one")

	sut.Add("two")
	t.assertLen(sut, 2)
	t.assertContains(sut, "o")
	t.assertContains(sut, "on")
	t.assertContains(sut, "one")
	t.assertContainsExpansion(sut, "one")
	t.assertContains(sut, "t")
	t.assertContains(sut, "tw")
	t.assertContains(sut, "two")
	t.assertContainsExpansion(sut, "two")

	sut.Add("three")
	t.assertLen(sut, 3)
	t.assertContains(sut, "o")
	t.assertContains(sut, "on")
	t.assertContains(sut, "one")
	t.assertContainsExpansion(sut, "one")
	t.assertContains(sut, "t")
	t.assertContains(sut, "tw")
	t.assertContains(sut, "two")
	t.assertContainsExpansion(sut, "two")
	t.assertContains(sut, "t")
	t.assertContains(sut, "th")
	t.assertContains(sut, "thr")
	t.assertContains(sut, "thre")
	t.assertContains(sut, "three")

	sut.Remove("one")
	t.assertLen(sut, 2)
	t.assertNotContainsExpansion(sut, "one")
	t.assertNotContains(sut, "o")
	t.assertNotContains(sut, "on")
	t.assertNotContains(sut, "one")

	sut.Remove("three")
	t.assertLen(sut, 1)
	t.assertNotContainsExpansion(sut, "three")
	t.assertNotContains(sut, "th")
	t.assertNotContains(sut, "thr")
	t.assertNotContains(sut, "thre")
	t.assertNotContains(sut, "three")

	sut.Remove("two")
	t.assertLen(sut, 0)
	t.assertNotContainsExpansion(sut, "two")

	sut.Add("1")
	sut.Add("12")
	sut.Add("123")
	sut.Add("1234")
	t.assertLen(sut, 4)
	t.assertContainsExpansion(sut, "1")
	t.assertContainsExpansion(sut, "12")
	t.assertContainsExpansion(sut, "123")
	t.assertContainsExpansion(sut, "1234")
	t.assertContains(sut, "1")
	t.assertContains(sut, "12")
	t.assertContains(sut, "123")
	t.assertContains(sut, "1234")

	sut.Remove("1")
	t.assertLen(sut, 3)
	t.assertNotContainsExpansion(sut, "1")
	t.assertContainsExpansion(sut, "12")
	t.assertContainsExpansion(sut, "123")
	t.assertContainsExpansion(sut, "1234")
	t.assertContains(sut, "1")
	t.assertContains(sut, "12")
	t.assertContains(sut, "123")
	t.assertContains(sut, "1234")
}

func TestAbbreviateOrphan(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"one",
	)

	expectedContains := map[string]string{
		"one": "o",
	}

	for e, c := range expectedContains {
		if ca := sut.Abbreviate(e); ca != c {
			t.Errorf("%q: expected shorted length %q but got %q", e, c, ca)
		}
	}
}

func TestAbbreviateDegenerate(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"mewtwo",
		"mew",
	)

	expectedContains := map[string]string{
		"mewtwo": "mewt",
		"mew":    "mew",
	}

	for e, c := range expectedContains {
		if ca := sut.Abbreviate(e); ca != c {
			ui.Log().Printf("%#v", sut)
			t.Errorf("%q: expected shorted length %q but got %q", e, c, ca)
		}
	}
}

func TestExpandDegenerate(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"mewtwo",
		"mew",
	)

	expectedContains := map[string]string{
		"mewt": "mewtwo",
		"mew":  "mew",
	}

	for e, c := range expectedContains {
		if ca := sut.Expand(e); ca != c {
			ui.Log().Printf("%#v", sut)
			t.Errorf("%q: expected expanded %q but got %q", e, c, ca)
		}
	}
}

func TestAbbreviate(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"12",
		"121",
		"123456",
		"127",
		"128",
		"654321",
		"mew",
		"mewtwo",
	)

	expectedContains := map[string]string{
		"12":       "12",
		"121":      "121",
		"123":      "123",
		"123456":   "123",
		"1234567":  "1234",
		"12345678": "1234",
		"124":      "124",
		"2":        "2",
		"mew":      "mew",
		"mewtwo":   "mewt",
	}

	for e, c := range expectedContains {
		if ca := sut.Abbreviate(e); ca != c {
			ui.Log().Print(t, "%#v", sut)
			t.Errorf("%q: expected shorted length %q but got %q", e, c, ca)
		}
	}
}

func TestExpandOrphan(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"654321",
	)

	expectedContains := map[string]string{
		"6":      "654321",
		"654321": "654321",
	}

	for a, e := range expectedContains {
		if ca := sut.Expand(a); ca != e {
			t.Errorf("%q: expected expanded %q but got %q", e, e, ca)
		}
	}
}

func TestExpand(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := Make(
		"12",
		"121",
		"127",
		"128",
		"123456",
		"654321",
	)

	expectedContains := map[string]string{
		"6":      "654321",
		"654321": "654321",
		"128":    "128",
		"123":    "123456",
		"1232":   "1232",
	}

	for a, e := range expectedContains {
		if ca := sut.Expand(a); ca != e {
			t.Errorf("%q: expected expanded %q but got %q", e, e, ca)
		}
	}
}

func TestDoesNotContainPrefix(t1 *testing.T) {
	t := t(ui.T{T: t1})
	makeSut := func() interfaces.TridexMutable {
		return Make(
			"121",
			"127",
			"128",
			"123456",
			"654321",
		)
	}

	sut := makeSut()
	e1 := "12"

	t.assertNotContainsExpansion(sut, e1)
	t.assertContains(sut, e1)
}

func TestRemove(t1 *testing.T) {
	t := t(ui.T{T: t1})

	makeSut := func() interfaces.TridexMutable {
		return Make(
			"12",
			"121",
			"127",
			"128",
			"123456",
			"654321",
		)
	}

	elements := []string{
		"12",
		"121",
		"127",
		"128",
		"123456",
		"654321",
	}

	for i, e := range elements {
		sut := makeSut()

		t.assertLen(sut, len(elements))

		sut.Remove(e)

		t.assertLen(sut, len(elements)-1)

		for j, e1 := range elements {
			if j == i {
				t.assertNotContainsExpansion(sut, e1)
			} else {
				t.assertContainsExpansion(sut, e1)
			}
		}
	}
}

func TestAll(t1 *testing.T) {
	testCases := [][]string{
		{
			"12",
			"121",
			"127",
			"128",
			"123456",
			"654321",
		},
		{
			"zz-archive",
			"zz-archive-recycle",
			"zz-archive-duplicate",
		},
		{
			"person-john",
			"person-eric",
			"zz-archive",
			"zz-archive-recycle",
			"zz-archive-duplicate",
		},
	}

	for i, tc := range testCases {
		t1.Run(
			fmt.Sprintf("test # %d", i),
			func(t1 *testing.T) {
				t := t(ui.T{T: t1})

				expected := tc

				sut := Make(expected...)

				actual := slices.Collect(sut.All())

				sort.Strings(expected)
				sort.Strings(actual)

				if !reflect.DeepEqual(expected, actual) {
					t.Errorf("expected %v, but got %v", expected, actual)
				}
			},
		)
	}
}

// func TestStructure(t1 *testing.T) {
// 	t := t(test_logz.T{T: t1})

//   els := []string{
//     "0",
//     "12",
//     "12-22",
//     "1_day",
//     "2022",
//     "2022-12-22",
//     "22",
//     "chore",
//     "pom",
//     "pom-0",
//     "urgency",
//     "urgency-1_day",
//     "w",
//     "w-2022",
//     "w-2022-12",
//     "w-2022-12-22",
//   }

// 	makeSut := func() schnittstellen.Tridex {
// 		return Make(els...)
// 	}

// 	elements := []string{
// 		"12",
// 		"121",
// 		"127",
// 		"128",
// 		"123456",
// 		"654321",
// 	}

// 	for i, e := range elements {
// 		sut := makeSut()

// 		t.assertLen(sut, len(elements))

// 		sut.Remove(e)

// 		t.assertLen(sut, len(elements)-1)

// 		for j, e1 := range elements {
// 			if j == i {
// 				t.assertNotContainsExpansion(sut, e1)
// 			} else {
// 				t.assertContainsExpansion(sut, e1)
// 			}
// 		}
// 	}
// }

// func TestShas(t *testing.T) {
// 	sut := Make(
// 		"7a2be8c643edd96b0cce2a1be32de30967a6db1f362047b954401458dd530f",
// 		"86324762e1fd27008f2b6c276ba82d3bbae349fd9819865a44931fae6bd41b",
// 		"905f0ed00a076771d48628477ff056e9b2e9d9ddcdafe4d78859517530f602",
// 		"a413bdb43426d528dc865d31e399719449b23481f5b9c10b8465a226fa863a",
// 		"b55eea6909f4b89967f531654dacb285723db6cd07d38f05de945f840db83d",
// 		"bc5784f13699736c709445114bf0374f9090c715549a826a2ad81a97ce055d",
// 		"be682a0034e409f23404b6588a7f39be6fe6eed333f341c9b97677ffa90303",
// 		"d123661f09fe5bd09d840b44c118c5c32cb8d0f33d1b9284aca95afdf18da9",
// 		"d45e1e12fececfc8613fa33bd237a5e511944234cb634ac05159b62438d15a",
// 		"e66bbf9a57566858d63522d83f4a3e17d7bdd8544c59d5c410be355b08b96a",
// 		"ec3f2668d283201e64db299bbdcc79e0885467172ea53246fef3c8880db14a",
// 		"ec5983b7f7d86cab238db64d0e8ea266dbbe3bf9133d4aa2acb61328abf94b",
// 		"f66f17833d028ca4aa311474a2e63f19ee6b46e379066defaf356f2a5405c8",
// 		"f8ff837c4d072cc7e28da224df391126952f6b6a6af237cdae00c663228c61",
// 		"ff05ec4214ad07c3f5f53ca6ed292a6115486f6864bb0612fa0e2d5fce7bec",
// 	)

// 	expectedContains := map[string]string{
// 		"6":    "654321",
// 		"128":  "128",
// 		"123":  "123456",
// 		"1232": "",
// 	}

// 	b, _ := json.Marshal(sut)

// 	for a, e := range expectedContains {
// 		if ca := sut.Expand(a); ca != e {
// 			test_logz.Errorf(t, "%#v: expected expanded %q but got %q", string(b), e,
// ca)
// 		}
// 	}
// }

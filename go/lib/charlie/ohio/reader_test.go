package ohio

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
)

func TestReaderIterateOneHappy(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "test string"
	test_value := values.MakeString(input)
	sut := MakeLineReaderIterate(
		test_value.Match,
	)

	t.AssertNoError(sut(input))
	t.AssertNoError(sut(input))
}

func TestReaderIterateOneSad(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "test string\n"
	test_value := values.MakeString("test string")
	sut := MakeLineReaderIterate(
		test_value.Match,
	)

	t.AssertError(sut(input))
	t.AssertError(sut(input))
}

func TestReaderIterateTwoHappy(t1 *testing.T) {
	t := ui.T{T: t1}

	test_value := values.MakeString("test string")
	test_value_two := values.MakeString("test string two")
	sut := MakeLineReaderIterate(
		test_value.Match,
		test_value_two.Match,
	)

	t.AssertNoError(sut("test string"))
	t.AssertNoError(sut("test string two"))
}

func TestReaderIterateTwoSad(t1 *testing.T) {
	t := ui.T{T: t1}

	test_value := values.MakeString("test string")
	test_value_two := values.MakeString("test string two")
	sut := MakeLineReaderIterate(
		test_value.Match,
		test_value_two.Match,
	)

	t.AssertNoError(sut("test string"))
	t.AssertError(sut("test string two x"))
}

func TestReaderKeyValueHappy(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := MakeLineReaderKeyValues(
		map[string]interfaces.FuncSetString{
			"#": values.MakeString("bez").Match,
			"%": values.MakeString("com").Match,
			"-": values.MakeString("et").Match,
			"!": values.MakeString("typ").Match,
		},
	)

	t.AssertNoError(sut("# bez"))
	t.AssertNoError(sut("% com"))
	t.AssertError(sut("X fail"))
	t.AssertNoError(sut("- et"))
	t.AssertNoError(sut("! typ"))
}

package dagnabit

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestNATOLevelMapperHeight0(t1 *testing.T) {
	t := ui.MakeT(t1)
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(0)
	t.AssertNoError(err)

	t.AssertEqualStrings("0", name)
}

func TestNATOLevelMapperHeight1(t1 *testing.T) {
	t := ui.MakeT(t1)
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(1)
	t.AssertNoError(err)

	t.AssertEqualStrings("alfa", name)
}

func TestNATOLevelMapperMaxHeight(t1 *testing.T) {
	t := ui.MakeT(t1)
	m := MakeNATOLevelMapper()

	name, err := m.LevelName(26)
	t.AssertNoError(err)

	t.AssertEqualStrings("zulu", name)
}

func TestNATOLevelMapperOutOfRange(t1 *testing.T) {
	t := ui.MakeT(t1)
	m := MakeNATOLevelMapper()

	_, err := m.LevelName(27)
	t.AssertError(err)
}

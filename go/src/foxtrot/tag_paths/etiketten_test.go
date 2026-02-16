package tag_paths

import (
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
)

func TestAddPaths(t1 *testing.T) {
	t := ui.T{T: t1}

	var es Tags

	areaHome, _ := catgut.MakeFromString("area-home")
	projectReno, _ := catgut.MakeFromString("project-reno")
	es.AddPath(MakePathWithType(
		areaHome,
		projectReno,
	))

	{
		area, _ := catgut.MakeFromString("area")
		i, ok := es.All.ContainsTag(area)

		if !ok {
			t.Errorf("expected some tag: %d, %t, %s", i, ok, es)
		}
	}

	areaCareer, _ := catgut.MakeFromString("area-career")
	projectRecurse, _ := catgut.MakeFromString("project-recurse")
	es.AddPath(MakePathWithType(
		areaCareer,
		projectRecurse,
	))

	{
		area, _ := catgut.MakeFromString("area")
		i, ok := es.All.ContainsTag(area)

		if !ok {
			t.Errorf("expected some tag: %d, %t, %s", i, ok, es.All)
		}
	}
}

func TestRealWorld(t1 *testing.T) {
	t := ui.T{T: t1}

	var es Tags

	pom1, _ := catgut.MakeFromString("pom-1")
	es.AddTag(pom1)
	reqCompInternet, _ := catgut.MakeFromString("req-comp-internet")
	es.AddTag(reqCompInternet)
	todayInProgress, _ := catgut.MakeFromString("today-in_progress")
	es.AddTag(todayInProgress)

	{
		e, _ := catgut.MakeFromString("req-comp-internet")
		_, ok := es.All.ContainsTag(e)

		if !ok {
			t.Errorf("expected %s to be in %s", e, es)
		}
	}

	project2022Recurse, _ := catgut.MakeFromString("project-2022-recurse")
	project24q2TalentShow, _ := catgut.MakeFromString("project-24q2-talent_show")
	es.AddPath(MakePathWithType(
		project2022Recurse,
		project24q2TalentShow,
	))

	e, _ := catgut.MakeFromString("req-comp-internet")
	_, ok := es.All.ContainsTag(e)

	if !ok {
		t.Errorf("expected %s to be in %s", e, es)
	}
}

func BenchmarkMatchFirstYes(b *testing.B) {
	var es Tags

	areaHome, _ := catgut.MakeFromString("area-home")
	projectReno, _ := catgut.MakeFromString("project-reno")
	es.AddPath(MakePathWithType(
		areaHome,
		projectReno,
	))

	areaCareer, _ := catgut.MakeFromString("area-career")
	projectRecurse, _ := catgut.MakeFromString("project-recurse")
	es.AddPath(MakePathWithType(
		areaCareer,
		projectRecurse,
	))

	m, _ := catgut.MakeFromString("area")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		es.All.ContainsTag(m)
	}
}

func BenchmarkMatchFirstNo(b *testing.B) {
	var es Tags

	areaHome, _ := catgut.MakeFromString("area-home")
	projectReno, _ := catgut.MakeFromString("project-reno")
	es.AddPath(MakePathWithType(
		areaHome,
		projectReno,
	))

	areaCareer, _ := catgut.MakeFromString("area-career")
	projectRecurse, _ := catgut.MakeFromString("project-recurse")
	es.AddPath(MakePathWithType(
		areaCareer,
		projectRecurse,
	))

	m, _ := catgut.MakeFromString("x")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		es.All.ContainsTag(m)
	}
}

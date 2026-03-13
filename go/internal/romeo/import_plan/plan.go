package import_plan

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
)

type Plan struct {
	Entries     []Entry
	SourcePaths []string
	HasErrors   bool
}

func (plan *Plan) CountByClassification() map[Classification]int {
	counts := make(map[Classification]int)

	for i := range plan.Entries {
		counts[plan.Entries[i].Classification]++
	}

	return counts
}

func (plan *Plan) CommittableCount() int {
	count := 0

	for i := range plan.Entries {
		if plan.Entries[i].Classification.IsCommittable() {
			count++
		}
	}

	return count
}

func (plan *Plan) TypeCount() int {
	count := 0

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		if genres.Make(entry.object.GetGenre()) == genres.Type {
			count++
		}
	}

	return count
}

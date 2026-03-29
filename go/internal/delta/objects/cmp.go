package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

var (
	EqualerSansTai               equaler
	Equaler                      = equaler{includeTai: true}
	EqualerSansTaiIncludeVirtual = equaler{includeVirtual: true}
)

type equaler struct {
	includeVirtual bool
	includeTai     bool
}

const debug = false

// TODO make better diffing facility
func (equaler equaler) Equals(a, b Metadata) bool {
	{
		a := a.(*metadata)
		b := b.(*metadata)

		if equaler.includeTai && !a.tai.Equals(b.tai) {
			if debug {
				ui.Debug().Print(&a.tai, "->", &b.tai)
			}
			return false
		}

		if !markl.Equals(&a.digBlob, &b.digBlob) {
			if debug {
				ui.Debug().Print(&a.digBlob, "->", &b.digBlob)
			}
			return false
		}

		if !markl.LockEquals(a.GetTypeLock(), b.GetTypeLock()) {
			if debug {
				ui.Debug().Print(&a.typ, "->", &b.typ)
			}
			return false
		}

		aTags := a.GetTags()
		bTags := b.GetTags()

		found := false
		for aTag := range aTags.All() {
			if (!equaler.includeVirtual && aTag.IsVirtual()) || aTag.IsEmpty() {
				continue
			}

			if !bTags.ContainsKey(bTags.Key(aTag)) {
				if debug {
					ui.Debug().Print(aTag, "-> X")
				}
				found = true
				break
			}
		}
		if found {
			if debug {
				ui.Debug().Print(aTags, "->", bTags)
			}

			return false
		}

		found2 := false
		for bTag := range bTags.All() {
			if !equaler.includeVirtual && bTag.IsVirtual() {
				continue
			}

			if !aTags.ContainsKey(aTags.Key(bTag)) {
				found2 = true
				break
			}
		}
		if found2 {
			if debug {
				ui.Debug().Print(aTags, "->", bTags)
			}
			return false
		}

		for aRef := range a.contents.AllReferences() {
			if _, ok := b.contents.getLock(aRef.String()); !ok {
				return false
			}
		}

		for bRef := range b.contents.AllReferences() {
			if _, ok := a.contents.getLock(bRef.String()); !ok {
				return false
			}
		}

		if !a.description.Equals(b.description) {
			if debug {
				ui.Debug().Print(a.description, "->", b.description)
			}
			return false
		}

		return true
	}
}

var Lessor lessor

type lessor struct{}

func (lessor) Less(a, b *metadata) bool {
	return a.tai.Less(b.tai)
}

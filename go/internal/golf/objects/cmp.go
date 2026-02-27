package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
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

		if equaler.includeTai && !a.Tai.Equals(b.Tai) {
			if debug {
				ui.Debug().Print(&a.Tai, "->", &b.Tai)
			}
			return false
		}

		if !markl.Equals(&a.DigBlob, &b.DigBlob) {
			if debug {
				ui.Debug().Print(&a.DigBlob, "->", &b.DigBlob)
			}
			return false
		}

		if !markl.LockEquals(a.GetTypeLock(), b.GetTypeLock()) {
			if debug {
				ui.Debug().Print(&a.Type, "->", &b.Type)
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

		if !a.Description.Equals(b.Description) {
			if debug {
				ui.Debug().Print(a.Description, "->", b.Description)
			}
			return false
		}

		return true
	}
}

var Lessor lessor

type lessor struct{}

func (lessor) Less(a, b *metadata) bool {
	return a.Tai.Less(b.Tai)
}

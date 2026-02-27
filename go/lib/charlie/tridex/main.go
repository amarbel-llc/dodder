package tridex

import (
	"encoding/gob"
	"sort"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

func init() {
	gob.Register(&Tridex{})
}

// TODO-P4 make generic
// TODO-P4 recycle nodes
// TODO-P4 confirm JSON structure is correct
// TODO switch to runes and rune readers as input
type Tridex struct {
	lock sync.RWMutex
	Root node
}

var _ interfaces.TridexMutable = &Tridex{}

func Make(vs ...string) (t interfaces.TridexMutable) {
	t = &Tridex{
		Root: node{
			Children: make(map[byte]node),
			IsRoot:   true,
		},
	}

	vs1 := make([]string, len(vs))
	copy(vs1, vs)

	sort.Slice(vs1, func(i, j int) bool { return len(vs1[i]) > len(vs1[j]) })

	for _, v := range vs1 {
		t.Add(v)
	}

	return t
}

func (tridex *Tridex) MutableClone() (b interfaces.TridexMutable) {
	ui.TodoP4("improve the performance of this")
	ui.TodoP4("collections-copy")
	ui.TodoP4("collections-reset")
	ui.TodoP4("collections-recycle")

	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	b = &Tridex{
		Root: tridex.Root.Copy(),
	}

	return b
}

func (tridex *Tridex) Len() int {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	return tridex.Root.Count
}

func (tridex *Tridex) ContainsAbbreviation(v string) bool {
	return tridex.Contains(v)
}

func (tridex *Tridex) Contains(v string) bool {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	return tridex.Root.Contains(v)
}

func (tridex *Tridex) ContainsExpansion(v string) bool {
	return tridex.ContainsExactly(v)
}

func (tridex *Tridex) ContainsExactly(v string) bool {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	return tridex.Root.ContainsExactly(v)
}

func (tridex *Tridex) Abbreviate(v string) string {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	return tridex.Root.Abbreviate(v, 0)
}

func (tridex *Tridex) Expand(v string) string {
	tridex.lock.RLock()
	defer tridex.lock.RUnlock()

	sb := &strings.Builder{}
	ok := tridex.Root.Expand(v, sb)

	if ok {
		return sb.String()
	} else {
		return v
	}
}

func (tridex *Tridex) Remove(v string) {
	tridex.lock.Lock()
	defer tridex.lock.Unlock()

	tridex.Root.Remove(v)
}

func (tridex *Tridex) Add(v string) {
	tridex.lock.Lock()
	defer tridex.lock.Unlock()

	if tridex.Root.ContainsExactly(v) {
		return
	}

	tridex.Root.Add(v)
}

func (tridex *Tridex) Any() string {
	var value string
	for value = range tridex.All() {
		break
	}

	return value
}

func (tridex *Tridex) All() interfaces.Seq[string] {
	return func(yield func(string) bool) {
		tridex.lock.Lock()
		defer tridex.lock.Unlock()

		for value := range tridex.Root.allWithAcc("") {
			if !yield(value) {
				return
			}
		}
	}
}

// TODO-P2 add Each and EachPtr methods
// func (t Tridex) GobEncode() (by []byte, err error) {
// 	bu := &bytes.Buffer{}
// 	enc := gob.NewEncoder(bu)
// 	err = enc.Encode(t.Root)
// 	by = bu.Bytes()
// 	return
// }

// func (t *Tridex) UnmarshalJSON(b []byte) error {
// 	bu := bytes.NewBuffer(b)
// 	dec := json.NewDecoder(bu)
// 	return dec.Decode(&t.Root)
// }

// func (t Tridex) MarshalJSON() (by []byte, err error) {
// 	bu := &bytes.Buffer{}
// 	enc := json.NewEncoder(bu)
// 	err = enc.Encode(t.Root)
// 	by = bu.Bytes()
// 	return
// }

// func (t *Tridex) GobDecode(b []byte) error {
// 	bu := bytes.NewBuffer(b)
// 	dec := gob.NewDecoder(bu)
// 	return dec.Decode(&t.Root)
// }

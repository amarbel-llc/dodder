package tridex

import (
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type node struct {
	Count            int
	Children         map[byte]node
	Value            string
	IsRoot           bool
	IncludesTerminus bool
}

func (noad *node) Add(value string) {
	if len(value) == 0 {
		noad.IncludesTerminus = true
		return
	}

	if value != noad.Value {
		noad.Count += 1
	}

	if noad.Count == 1 {
		noad.Value = value
		return
	} else if noad.Value != "" && noad.Value != value {
		noad.Add(noad.Value)
		noad.Value = ""
	}

	c := value[0]

	var child node
	ok := false
	child, ok = noad.Children[c]

	if !ok {
		child = node{Children: make(map[byte]node)}
	}

	child.Add(value[1:])

	noad.Children[c] = child
}

func (noad *node) Remove(value string) {
	if value == "" {
		noad.Count -= 1
		noad.IncludesTerminus = false
		return
	}

	if noad.Value == value {
		noad.Count -= 1
		noad.Value = ""
		return
	}

	first := value[0]

	rest := ""

	if len(value) > 1 {
		rest = value[1:]
	}

	child, ok := noad.Children[first]

	if ok {
		child.Remove(rest)
		noad.Count -= 1

		if child.Count == 0 {
			delete(noad.Children, first)
		} else {
			noad.Children[first] = child
		}
	}
}

func (noad node) Contains(value string) (ok bool) {
	if len(value) == 0 {
		ok = true
		return ok
	}

	if noad.Count == 1 && noad.Value != "" {
		ok = strings.HasPrefix(noad.Value, value)
		return ok
	}

	c := value[0]

	var child node

	child, ok = noad.Children[c]

	if ok {
		ok = child.Contains(value[1:])
	}

	return ok
}

func (noad node) ContainsExactly(value string) (ok bool) {
	if len(value) == 0 {
		ok = noad.IncludesTerminus
		return ok
	}

	if noad.Value != "" {
		ok = noad.Value == value
		return ok
	}

	c := value[0]

	var child node

	child, ok = noad.Children[c]

	if ok {
		ok = child.ContainsExactly(value[1:])
	}

	return ok
}

func (noad node) Any() byte {
	for c := range noad.Children {
		return c
	}

	return 0
}

func (noad node) Expand(
	value string,
	stringBuilder *strings.Builder,
) (ok bool) {
	var c byte
	var rem string

	if len(value) == 0 {
		switch noad.Count {

		case 0:
			return true

		case 1:
			if !noad.IncludesTerminus {
				stringBuilder.WriteString(noad.Value)
			}

			return true
		}
	} else {
		switch noad.Count {
		case 1:
			ok = strings.HasPrefix(noad.Value, value)

			if ok {
				stringBuilder.WriteString(noad.Value)
			}

			return ok

		default:
			rem = value[1:]
			c = value[0]
		}
	}

	var child node

	if child, ok = noad.Children[c]; ok {
		stringBuilder.WriteByte(c)
		return child.Expand(rem, stringBuilder)
	}

	return ok
}

func (noad node) Abbreviate(value string, loc int) string {
	if noad.IsRoot && len(noad.Children) == 0 {
		if noad.Value != "" {
			return noad.Value[0:1]
		} else {
			return ""
		}
	}

	if len(value)-1 < loc {
		return value
	}

	if noad.Count == 1 && noad.ContainsExactly(value[loc:]) &&
		!noad.IncludesTerminus {
		return value[0:loc]
	}

	c := value[loc]

	child, ok := noad.Children[c]

	if ok {
		return child.Abbreviate(value, loc+1)
	} else {
		if len(value)-1 < loc {
			return value
		} else {
			return value[0 : loc+1]
		}
	}
}

func (noad *node) Copy() (b node) {
	b = *noad

	for i, c := range b.Children {
		b.Children[i] = c.Copy()
	}

	return b
}

func (noad *node) allWithAcc(acc string) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		if noad.Value != "" {
			if !yield(acc + noad.Value) {
				return
			}
		}

		if noad.IncludesTerminus {
			if !yield(acc) {
				return
			}
		}

		for char, child := range noad.Children {
			for value := range child.allWithAcc(acc + string(char)) {
				if !yield(value) {
					return
				}
			}
		}
	}
}

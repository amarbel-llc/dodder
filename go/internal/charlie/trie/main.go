package trie

import (
	"bytes"
	"encoding/gob"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
)

type Trie struct {
	root node
}

type node struct {
	Count    int
	Children map[byte]node
	IsRoot   bool
}

func Make(vs ...string) (t *Trie) {
	t = &Trie{
		root: node{
			Children: make(map[byte]node, 255),
			IsRoot:   true,
		},
	}

	for _, v := range vs {
		t.Add(v)
	}

	return t
}

func (t *Trie) Contains(v string) bool {
	return t.root.Contains(v)
}

func (t *Trie) Abbreviate(v string) string {
	return t.root.Abbreviate(v, 0)
}

func (t *Trie) Expand(v string) string {
	sb := &strings.Builder{}
	ok := t.root.Expand(v, sb)

	if ok {
		return sb.String()
	} else {
		return ""
	}
}

func (t *Trie) Add(v string) {
	if t.Contains(v) {
		return
	}

	t.root.Add(v)
}

func (n *node) Add(v string) {
	if len(v) == 0 {
		return
	}

	n.Count += 1

	c := v[0]

	var child node
	ok := false
	child, ok = n.Children[c]

	if !ok {
		child = node{Children: make(map[byte]node)}
	}

	child.Add(v[1:])

	n.Children[c] = child
}

func (n node) Contains(v string) bool {
	if len(v) == 0 {
		return true
	}

	c := v[0]

	child, ok := n.Children[c]

	if ok {
		return child.Contains(v[1:])
	} else {
		return false
	}
}

func (n node) Any() byte {
	for c := range n.Children {
		return c
	}

	return 0
}

func (n node) Expand(v string, sb *strings.Builder) (ok bool) {
	ui.Err().Printf("v: %q, sb %q", v, sb.String())

	var c byte
	var rem string

	if len(v) == 0 {
		switch n.Count {

		case 0:
			return true

		case 1:
			c = n.Any()
		}
	} else {
		rem = v[1:]
		c = v[0]
	}

	var child node

	if child, ok = n.Children[c]; ok {
		sb.WriteByte(c)
		return child.Expand(rem, sb)
	}

	return ok
}

func (n node) Abbreviate(v string, loc int) string {
	if n.IsRoot && len(n.Children) == 0 {
		return ""
	}

	if len(v)-1 < loc {
		return v
	}

	if n.Count == 1 && n.Contains(v[loc:]) {
		return v[0:loc]
	}

	c := v[loc]

	child, ok := n.Children[c]

	if ok {
		return child.Abbreviate(v, loc+1)
	} else {
		if len(v)-1 < loc {
			return v
		} else {
			return v[0 : loc+1]
		}
	}
}

func (t Trie) GobEncode() (by []byte, err error) {
	bu := &bytes.Buffer{}
	enc := gob.NewEncoder(bu)
	err = enc.Encode(t.root)
	by = bu.Bytes()
	return by, err
}

func (t *Trie) GobDecode(b []byte) error {
	bu := bytes.NewBuffer(b)
	dec := gob.NewDecoder(bu)
	return dec.Decode(&t.root)
}

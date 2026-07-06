package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
)

//go:generate tommy generate
type V1 struct {
	Side      Side     `toml:"side"`
	Tai       ids.Tai  `toml:"tai"`
	MarklId   markl.Id `toml:"markl-id"`
	WordCount int      `toml:"word-count"`
}

func (v V1) GetSide() Side {
	return v.Side
}

func (v V1) GetTai() ids.Tai {
	return v.Tai
}

func (v V1) GetMarklId() markl.Id {
	return v.MarklId
}

func (v V1) GetWordCount() int {
	return v.WordCount
}

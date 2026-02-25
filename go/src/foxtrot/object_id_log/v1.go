package object_id_log

import (
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

var _ Entry = V1{}

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

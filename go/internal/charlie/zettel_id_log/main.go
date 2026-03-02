package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
)

type Side uint8

const (
	SideYin  Side = iota
	SideYang
)

type Entry interface {
	GetSide() Side
	GetTai() ids.Tai
	GetMarklId() markl.Id
	GetWordCount() int
}

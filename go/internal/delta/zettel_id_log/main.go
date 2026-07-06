package zettel_id_log

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	charlie_zil "code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
)

type (
	Side = charlie_zil.Side
	V1   = charlie_zil.V1
)

const (
	SideYin  = charlie_zil.SideYin
	SideYang = charlie_zil.SideYang
)

type Entry interface {
	GetSide() Side
	GetTai() ids.Tai
	GetMarklId() markl.Id
	GetWordCount() int
}

var _ Entry = V1{}

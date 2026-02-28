package ids

import (
	"code.linenisgreat.com/dodder/go/internal/_/doddish"
)

type StringerSansRepo struct {
	Id Id
}

func (stringer *StringerSansRepo) String() string {
	switch objectId := stringer.Id.(type) {
	case *ObjectId:
		return objectId.StringSansRepo()

	default:
		return objectId.String()
	}
}

type StringerSansOp struct {
	Id Id
}

func (stringer StringerSansOp) String() string {
	seq := stringer.Id.ToSeq()

	switch {
	case seq.MatchStart(
		doddish.TokenMatcherOp(doddish.OpPathSeparator),
	):
		return seq[1:].String()

	default:
		return seq.String()
	}
}

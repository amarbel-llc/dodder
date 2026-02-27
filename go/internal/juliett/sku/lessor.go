package sku

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
)

func TransactedCompare(left, right *Transacted) cmp.Result {
	if TransactedLessor.Less(left, right) {
		return cmp.Less
	} else if TransactedEqualer.Equals(left, right) {
		return cmp.Equal
	} else {
		return cmp.Greater
	}
}

type transactedLessorTaiOnly struct{}

func (transactedLessorTaiOnly) Less(a, b *Transacted) bool {
	return a.GetTai().Less(b.GetTai())
}

func (transactedLessorTaiOnly) LessPtr(a, b *Transacted) bool {
	return a.GetTai().Less(b.GetTai())
}

type transactedLessorStable struct{}

func (transactedLessorStable) Less(a, b *Transacted) bool {
	if result := a.GetTai().SortCompare(b.GetTai()); !result.IsEqual() {
		return result.IsLess()
	}

	return a.GetObjectId().String() < b.GetObjectId().String()
}

func (transactedLessorStable) LessPtr(a, b *Transacted) bool {
	return a.GetTai().Less(b.GetTai())
}

type transactedEqualer struct{}

func (transactedEqualer) Equals(a, b *Transacted) bool {
	return a.Equals(b)
}

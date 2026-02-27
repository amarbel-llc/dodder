package cmp

type Result interface {
	cmp()

	GetCompareInt() int
	IsLess() bool
	IsEqual() bool
	IsGreater() bool
}

const (
	Less    = result(-1)
	Equal   = result(0)
	Greater = result(1)
)

//go:generate stringer -type=result
type result int

var _ Result = result(0)

func (result) cmp() {}

func (result result) GetCompareInt() int {
	return int(result)
}

func (result result) IsLess() bool {
	return result == Less
}

func (result result) IsEqual() bool {
	return result == Equal
}

func (result result) IsGreater() bool {
	return result == Greater
}

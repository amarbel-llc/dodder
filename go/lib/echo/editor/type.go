package editor

//go:generate stringer -type=Type
type Type byte

const (
	TypeUnknown = Type(iota)
	TypeVim
)

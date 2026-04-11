package object_change_type

//go:generate stringer -type=Type
type Type byte

const (
	TypeUnknown = Type(iota)
	TypeLatest  = Type(1 << iota)
	TypeHistorical
)

func (a *Type) Add(bs ...Type) {
	for _, b := range bs {
		*a |= b
	}
}

func (a *Type) Del(b Type) {
	*a &= ^b
}

func (a Type) Contains(b Type) bool {
	return a&b != 0
}

func (a Type) ContainsAny(bs ...Type) bool {
	for _, b := range bs {
		if a.Contains(b) {
			return true
		}
	}

	return false
}

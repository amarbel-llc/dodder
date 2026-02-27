package descriptions

var Equaler equaler

type equaler struct{}

func (equaler) Equals(a, b Description) bool {
	return a.value == b.value
}

func (equaler) EqualsPtr(a, b *Description) bool {
	return a.value == b.value
}

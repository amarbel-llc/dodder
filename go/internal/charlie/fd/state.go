package fd

type stateTypeSigil struct{}

type State interface {
	isState() stateTypeSigil
}

type state int

func (state) isState() stateTypeSigil {
	return stateTypeSigil{}
}

const (
	StateUnknown = state(iota)
	StateFileInfo
	StateRead
	StateStored
)

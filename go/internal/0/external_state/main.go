package external_state

import "fmt"

type State int

const (
	Unknown = State(iota)
	Tracked
	Untracked
	Recognized
)

func (s State) String() string {
	switch s {
	case Tracked:
		return "tracked"

	case Untracked:
		return "untracked"

	case Recognized:
		return "recognized"

	default:
		return fmt.Sprintf("Unknown(%d)", s)
	}
}

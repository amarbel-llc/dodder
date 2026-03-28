package zettel_id_log

import (
	"fmt"
)

type Side uint8

const (
	SideYin Side = iota
	SideYang
)

func (s Side) String() string {
	switch s {
	case SideYin:
		return "yin"
	case SideYang:
		return "yang"
	default:
		return fmt.Sprintf("side(%d)", s)
	}
}

func (s Side) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

func (s *Side) UnmarshalText(text []byte) error {
	switch string(text) {
	case "yin":
		*s = SideYin
	case "yang":
		*s = SideYang
	default:
		return fmt.Errorf("unknown zettel id log side: %q", text)
	}

	return nil
}

package collections

import (
	"fmt"
	"strings"
)

type KeyFunc[T any] func(T) string

func MakeKey(ss ...fmt.Stringer) string {
	sb := &strings.Builder{}

	for i, s := range ss {
		if s == nil {
			continue
		}

		v := s.String()

		if v == "" {
			continue
		}

		if i > 0 {
			sb.WriteString(".")
		}

		sb.WriteString(s.String())
	}

	return sb.String()
}

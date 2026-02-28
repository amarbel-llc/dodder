package object_id_provider

import "strings"

func Clean(v string) string {
	v = strings.ToLower(v)
	v = strings.Map(
		func(r rune) rune {
			if r > 'z' {
				return -1
			}

			return r
		},
		v,
	)

	v = strings.TrimSpace(v)

	return v
}

package queries

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

func (query *Query) StringDebug() string {
	var sb strings.Builder

	if query.defaultQuery != nil {
		fmt.Fprintf(&sb, "default: %q", query.defaultQuery)
	}

	first := true

	for _, g := range query.sortedUserQueries() {
		if !first {
			sb.WriteRune(' ')
		}

		sb.WriteString(g.StringDebug())

		first = false
	}

	sb.WriteString(" | ")
	first = true

	for _, g := range genres.All() {
		q, ok := query.optimizedQueries[g]

		if !ok {
			continue
		}

		if !first {
			sb.WriteRune(' ')
		}

		sb.WriteString(q.String())

		first = false
	}

	return sb.String()
}

func (query *Query) StringOptimized() string {
	var sb strings.Builder

	first := true

	// qg.FDs.Each(
	// 	func(f *fd.FD) error {
	// 		if !first {
	// 			sb.WriteRune(' ')
	// 		}

	// 		sb.WriteString(f.String())

	// 		first = false

	// 		return nil
	// 	},
	// )

	for _, g := range genres.All() {
		q, ok := query.optimizedQueries[g]

		if !ok {
			continue
		}

		if !first {
			sb.WriteRune(' ')
		}

		sb.WriteString(q.String())

		first = false
	}

	return sb.String()
}

package tag_paths

import (
	"fmt"
	"strings"
)

type TagWithParentsAndTypes struct {
	*Tag
	Parents PathsWithTypes
}

func (ewp TagWithParentsAndTypes) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s:%s", ewp.Tag, ewp.Parents)

	return sb.String()
}

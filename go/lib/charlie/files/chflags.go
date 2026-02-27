package files

import (
	"strings"
)

func isNotExists(err error, msg []byte) bool {
	return strings.HasSuffix(string(msg), "No such file or directory")
}

type userChangesOptions struct {
	allow     bool
	recursive bool
}

func SetAllowUserChangesRecursive(paths ...string) (err error) {
	return setUserChanges(
		paths,
		userChangesOptions{allow: true, recursive: true},
	)
}

func SetAllowUserChanges(paths ...string) (err error) {
	return setUserChanges(paths, userChangesOptions{allow: true})
}

func SetDisallowUserChanges(paths ...string) (err error) {
	return setUserChanges(paths, userChangesOptions{allow: false})
}

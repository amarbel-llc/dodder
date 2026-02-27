//go:build !chrest

package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/quebec/env_workspace"
)

func (local *Repo) initializeBrowserWorkspace() map[string]*env_workspace.Store {
	return nil
}

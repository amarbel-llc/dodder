//go:build chrest

package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/romeo/env_workspace"
	"code.linenisgreat.com/dodder/go/internal/sierra/store_browser"
)

func (local *Repo) initializeBrowserWorkspace() map[string]*env_workspace.Store {
	return map[string]*env_workspace.Store{
		"browser": {
			StoreLike: store_browser.Make(
				local.config,
				local.GetEnvRepo(),
				local.PrinterTransactedDeleted(),
			),
		},
	}
}

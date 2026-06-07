package command

type (
	Cmd interface {
		Run(Request)
	}

	Description struct {
		Short, Long string
	}

	CommandWithDescription interface {
		GetDescription() Description
	}

	// CommandWithResetCLIState is implemented by commands whose
	// flag-bound state accumulates across invocations when the same
	// registered command value is reused in a long-lived process.
	// One-shot CLI processes never need it; the MCP bridge calls
	// ResetCLIState before each invocation so accumulating Var-bound
	// values (descriptions.Description.Set appends, tag sets union)
	// don't leak between tool calls (#247). The method name is
	// deliberately NOT `Reset` — embedded fields like ids.RepoId
	// promote a `Reset` method with much narrower semantics, and an
	// interface check on `Reset()` would silently match those.
	CommandWithResetCLIState interface {
		ResetCLIState()
	}
)

# stack_frame

Stack frame capture for enhanced error context and debugging.

## Functions (go:noinline for accurate stack traces)

- `Wrap(err)`: Wrap error with current stack frame
- `WrapSkip(skip, err)`: Wrap error with stack frame skipping N frames
- `Wrapf(err, format, args)`: Wrap error with formatted message and stack frame
- `Errorf(format, args)`: Create new error with formatted message and stack frame

## Key Types

- `Frame`: Individual stack frame with file, line, and function info
- `ErrorAndFrame`, `ErrorsAndFrames`: Error with associated frame(s)
- `ErrorTree`: Hierarchical error structure (see error_tree.go)

Used by alfa/errors package for detailed error context with call stack information.
All wrapping functions are go:noinline to ensure accurate frame capture.

# errors

Enhanced error handling with stack traces, context, and error groups.

## Key Functions

- `Wrap/Wrapf()` - wraps errors with stack trace
- `WrapSkip()` - wraps with custom stack skip level
- `ErrorWithStackf()` - creates new error with stack
- `Join()` - combines multiple errors into Group
- `PanicIfError()` - converts errors to panics

## Key Types

- `Group` - error group implementing multi-error unwrap
- Wait group helpers for parallel/serial error collection

## Features

- Stack trace capture via stack_frame integration
- Debug/release build variants (main_debug.go/main_release.go)
- HTTP error utilities (http.go)
- Signal handling (signal.go)
- Deferred error handling (deferred.go)

## Notes

- Never wrap io.EOF (panics if attempted)
- WrapExceptSentinel variants for sentinel error handling

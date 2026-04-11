# exec

Command execution utility with variadic argument flattening.

## Functions

- `ExecCommand(command, ...args)`: Create exec.Cmd with flattened variadic string slice arguments

Simplifies building exec.Command calls when arguments come from multiple slices.

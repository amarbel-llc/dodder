# vim_cli_options_builder

Builder pattern for constructing Vim command-line arguments.

## Type

- `Builder`: Wrapper around []string with fluent builder methods

## Methods

- `New()`: Create empty builder
- `WithFileType(ft)`: Set file type with `-c "set ft=type"`
- `WithSourcedFile(path)`: Source a file with `-c "source path"`
- `WithCursorLocation(row, col)`: Position cursor with `-c "call cursor(row, col)"`
- `WithInsertMode()`: Start in insert mode with `-c "startinsert!"`
- `Build()`: Return final []string arguments

Fluent API for building Vim invocation arguments when opening files programmatically.

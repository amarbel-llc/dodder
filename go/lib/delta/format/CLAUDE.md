# format

Format writer utilities for type-safe I/O with composable writer combinators,
line-based reading/writing, and sequential write helpers.

## Key Functions

### Writer Combinators

- `MakeWriter[E]`: Create writer from format function and element
- `MakeWriterOr[A,B]`: Write first non-empty element (Stringer-based)
- `MakeWriterPtr[E]`: Writer for pointer types
- `MakeFormatString`: Printf-style formatted writer
- `MakeStringer`: Writer from any `fmt.Stringer`
- `MakeFormatStringer[E]`: Convert string formatter to writer
- `MakeFormatStringRightAligned`: Right-aligned formatted string
- `Write`: Sequentially execute multiple FuncWriters

## Line I/O

- `LineWriter`: Buffered line output with `WriteTo(io.Writer)`
  - `WriteLines()`, `WriteFormat()`, `WriteKeySpaceValue()`
  - `WriteEmpty()`, `WriteExactlyOneEmpty()`
- `MakeLineReaderConsumeEmpty`: Line reader skipping blanks
- `MakeLineReaderPassThruEmpty`: Line reader passing blanks
- `MakeDelimReaderConsumeEmpty`: Custom delimiter reader
- `ReadLines()`, `ReadSep()`: Convenience read functions

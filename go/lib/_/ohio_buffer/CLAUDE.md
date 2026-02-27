# ohio_buffer

Buffer operation utilities with error checking.

## Functions

- `Copy(dst, src)`: Copy bytes.Buffer contents with length validation
- `MakeErrLength()`: Create length mismatch error (defined in errors.go)

Wraps standard buffer operations with additional safety checks.

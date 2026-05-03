# ohio_files

File copying utilities with line-by-line transformation support.

## Functions

- `CopyFileWithTransform(src, dst, delim, transform)`: Copy file applying transform to each delimited segment
- `CopyFileLines(src, dst)`: Copy file line-by-line without transformation

## Implementation

- Uses buffered I/O from object pools for efficiency
- Handles EOF detection and proper resource cleanup
- Deferred error handling with proper flush semantics
- Transform function applied to each line/segment before writing

Used for file operations requiring per-line processing or validation.

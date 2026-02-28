# files

File operation utilities with error wrapping and various open modes.

## Key Functions

- `Open`, `Create`: Basic file operations with error wrapping
- `OpenExclusive*`: Exclusive lock variants (read/write/append)
- `OpenReadWrite`, `OpenCreate`: Create-if-not-exists variants
- `TryOrTimeout`: Retry file operations with timeout
- `TryOrMakeDirIfNecessary`: Auto-create parent directories
- `Exists`: Check file existence

## Features

- Consistent error wrapping for all operations
- Various file mode combinations
- Timeout-based retry logic
- Temporary file/directory creation via TemporaryFS
- Hash bucket path generation (Git-style bucketed paths)

## Temp FS

- `TemporaryFS`: Struct wrapping a base path for temp file/dir creation
  - `DirTemp()`, `DirTempWithTemplate()`: Create temp directories
  - `FileTemp()`, `FileTempWithTemplate()`: Create temp files

## Hash Bucket Paths

- `MakeHashBucketPath()`: Build paths with hash-based directory bucketing
- `MakeHashBucketPathJoinFunc()`: Curried version returning a join function
- `PathFromHeadAndTail()`: Build paths from head/tail string interface
- `MakeDirIfNecessary()`: Create parent directories and return joined path
- `MakeDirIfNecessaryForStringerWithHeadAndTail()`: Variant using head/tail

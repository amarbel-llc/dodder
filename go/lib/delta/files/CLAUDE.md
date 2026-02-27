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

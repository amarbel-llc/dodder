# primordial

Fundamental system-level utilities.

## Functions

- `IsTty(f)`: Check if file descriptor is a terminal (TTY)

Uses golang.org/x/term for cross-platform terminal detection.
Used for conditional output formatting based on terminal capabilities.

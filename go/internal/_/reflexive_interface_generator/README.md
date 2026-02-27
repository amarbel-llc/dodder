# Reflexive Interface Generator

A Go code generator that automatically creates interfaces from concrete types, similar to `stringer` but for generating interfaces with all public methods.

## Installation

```bash
go install .
```

## Usage

Add a `//go:generate` directive to your Go source file:

```go
//go:generate go run github.com/friedenberg/dodder/go/src/alfa/reflexive_interface_generator -type=YourType
```

Or run directly:

```bash
go run . -type=MyService
```

## Example

Given a type like:

```go
type MyService struct {
    name    string
    counter int
}

func (s *MyService) GetName() string { return s.name }
func (s *MyService) SetName(name string) { s.name = name }
func (s *MyService) Increment() int { s.counter++; return s.counter }
```

The generator will create:

```go
// IMyService is an interface that mirrors all methods of MyService.
type IMyService interface {
    GetName() string
    SetName(name string)
    Increment() int
}

// Compile-time check that MyService implements IMyService.
var _ IMyService = (*MyService)(nil)
```

## Features

- Automatically generates interfaces with all exported methods
- Adds `I` prefix to interface names
- Includes compile-time verification that the original type implements the interface
- Detects and imports required packages (e.g., `io`, `context`, `time`)
- Preserves method signatures exactly
- Supports multiple types via comma separation: `-type=Type1,Type2`
- Customizable output file name with `-output` flag

## Flags

- `-type`: Comma-separated list of type names (required)
- `-output`: Output file name (default: `<lowercase_type>_interface.go`)
- `-tags`: Comma-separated list of build tags

## Use Cases

- Creating testable interfaces from concrete implementations
- Generating mock interfaces for testing
- Retrofitting interfaces onto existing types
- Documenting the public API of a type

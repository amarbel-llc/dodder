# reflexive_interface_generator

Code generation tool for extracting interfaces from concrete types.

## Usage

`go run . -type=TypeName [-output=filename] [-tags=buildtags]`

## Functionality

- Parses Go source using go/packages, go/ast, and go/types
- Extracts all exported methods from specified type
- Generates interface definition mirroring the type's methods
- Adds compile-time assertion that type implements interface
- Runs goimports to fix imports automatically

## Generated Output

- Interface named `I{TypeName}` with all exported methods
- Compile-time verification: `var _ ITypeName = (*TypeName)(nil)`
- Properly formatted with comments and import statements

Used to create interface definitions from existing concrete types for dependency injection
and testing.

# flags

Custom flag parsing library extending Go's standard `flag` package with policy-based flag handling and custom error types.

## Key Types

- `FlagSet`: Extended flag set with policy support
- `Flag`: Flag definition with name, usage, value, and default
- `FlagWithPolicy`: Flag wrapper that includes parsing policy

## Usage

Provides the standard flag interface (`Bool`, `Int`, `String`, etc.) plus `Var` for custom flag values implementing `interfaces.FlagValue`.

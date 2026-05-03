# expansion

String expansion utilities for hierarchical identifiers.

## Key Types

- `Expander`: Interface for expanding strings into sequences
- `ExpanderRight`: Expands from the right using a delimiter
- `ExpanderAll`: Expands all parts of a delimited string

## Usage

Used to expand tag paths like `a-b-c` into `[a, a-b, a-b-c]` for hierarchical matching.

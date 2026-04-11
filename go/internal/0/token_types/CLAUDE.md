# token_types

Token type enumeration for query parsing and lexical analysis.

## Type

- `TokenType`: Enumeration of lexical token categories

## Constants

- `TypeIncomplete`: Incomplete/malformed token
- `TypeOperator`: Operators like `=`, `,`, `.`, `:`, `+`, `?`, `^`, `[`, `]`
- `TypeIdentifier`: IDs, tags, paths (e.g., "one/uno", "tag-one", "!type", "@sha")
- `TypeLiteral`: Quoted strings with escape support
- `TypeField`: Field assignments (e.g., field="value", url="text with \"escapes\"")

Uses go:generate stringer for string representation.
Used in query parsing for Zettel search and filtering operations.

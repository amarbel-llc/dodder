package token_types

/* Token types:
- id: [-!/a-z_.]+
- literal: ".*"

*/

//go:generate stringer -type=TokenType
type TokenType int

const (
	TypeIncomplete = TokenType(iota)
	TypeOperator   // " =,.:+?^[]"
	TypeIdentifier // ["one/uno", "tag-one", "!type", "/browser/bookmark-1", "@sha"...]
	TypeLiteral    // ["\"some text\"", "\"some text \\\" with escape\""]
	TypeField      // ["field=\"some text\"", "url=\"some text \\\" with escape\""]
)

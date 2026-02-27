package doddish

//go:generate stringer -type=TokenType
type TokenType byte

const (
	TokenTypeIncomplete = TokenType(iota)
	TokenTypeOperator   // " =,.:+?^[]"
	TokenTypeIdentifier // ["one", "uno", "tag", "one", "type", "/browser/bookmark-1", "sha"...]
	TokenTypeLiteral    // ["\"some text\"", "\"some text \\\" with escape\""]
)

func (expected TokenType) Match(actual Token) bool {
	return actual.Type == expected
}

// TODO use collections_slice
type TokenTypes []TokenType

// TODO use collections_slice
func (actual TokenTypes) Equals(expected ...TokenType) bool {
	if len(actual) != len(expected) {
		return false
	}

	for i, a := range actual {
		if a != expected[i] {
			return false
		}
	}

	return true
}

package doddish

type TokenMatcher interface {
	Match(Token) bool
}

type TokensMatcher []TokenMatcher

var (
	// @abcd
	TokenMatcherBlobDigest = TokensMatcher{
		TokenMatcherOp('@'),
		TokenTypeIdentifier,
	}

	// !key
	TokenMatcherType = TokensMatcher{
		TokenMatcherOp('!'),
		TokenTypeIdentifier,
	}

	// !key@abcd
	TokenMatcherTypeLock = TokensMatcher{
		TokenMatcherOp('!'),
		TokenTypeIdentifier,
		TokenMatcherOp('@'),
		TokenTypeIdentifier,
	}

	// key@abcd
	TokenMatcherDodderTag = TokensMatcher{
		TokenTypeIdentifier,
		TokenMatcherOp('@'),
		TokenTypeIdentifier,
	}

	// key=value
	TokenMatcherKeyValue = TokensMatcher{
		TokenTypeIdentifier,
		TokenMatcherOp(OpExact),
	}

	// key="value"
	TokenMatcherKeyValueLiteral = TokensMatcher{
		TokenTypeIdentifier,
		TokenMatcherOp(OpExact),
		TokenTypeLiteral,
	}

	TokenMatcherTai = TokensMatcher{
		TokenTypeIdentifier,
		TokenMatcherOp('.'),
		TokenTypeIdentifier,
	}
)

type TokenMatcherOp byte

func (tokenMatcher TokenMatcherOp) Match(token Token) bool {
	if token.Type != TokenTypeOperator {
		return false
	}

	if token.Contents[0] != byte(tokenMatcher) {
		return false
	}

	return true
}

func TokenMatcherOr(tm ...TokenMatcher) tokenMatcherOr {
	return tokenMatcherOr(tm)
}

type tokenMatcherOr []TokenMatcher

func (tokenMatcher tokenMatcherOr) Match(token Token) bool {
	for _, t := range tokenMatcher {
		if t.Match(token) {
			return true
		}
	}

	return false
}

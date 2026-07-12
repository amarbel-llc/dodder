; Highlights for the doddish query language (docs/man.7/doddish.md).

; tags -> Constant
(tag (identifier) @constant)
(dependent_tag "-" @operator)
(dependent_tag name: (identifier) @constant)

; type filters ( !name ) -> Type
(type_filter "!" @punctuation.special)
(type_filter name: (identifier) @type)

; exact match ( =predicate )
(exact "=" @operator)

; object ids ( left/right, /repo )
(object_id left: (identifier) @constant)
(object_id right: (identifier) @constant)
(object_id name: (identifier) @module)
(object_id "/" @punctuation.delimiter)

; sigils ( : + . ? ) -> operators
(sigil) @operator

; genres ( z t e b k r, comma-combinable ) -> qualifiers
(genre (identifier) @type.qualifier)
(genres "," @punctuation.delimiter)

; grouping and negation
(group ["[" "]"] @punctuation.bracket)
(negation "^" @keyword.operator)
(group_negation "^" @keyword.operator)

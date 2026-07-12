; Highlights for the organize-text format (docs/man.7/organize-text.md).
; Preserves the contract from zz-vim/syntax/dodder-organize.vim:
;   object id -> Identifier, type -> Type, field value -> Constant/String,
;   escape -> SpecialChar, description -> String, heading tag -> Title.

; --- hyphence metadata header (shared with the hyphence grammar) ---
(metadata "---" @punctuation.special)
(description_line "#" @markup.heading.marker)
(description_line text: (description_text) @markup.heading)
(tag_line "-" @punctuation.special)
(tag_line name: (tag_name) @constant)
(ref_line "<" @punctuation.special)
(ref_line ref: (reference_id) @constant)
(type_line "!" @punctuation.special)
(type_line type: (type_name) @type)
(comment_line "%" @comment)
(comment_line text: (comment_text) @comment)

; --- headings ( #, ##, ... tag, tag ) ---
(heading marker: (heading_marker) @markup.heading.marker)
(heading (heading_tag) @constant)

; --- object line prefixes ---
(object_line "-" @punctuation.special)
(object_line "%" @comment)
(object_line description: (description) @string)

; --- box interior ---
(box ["[" "]"] @punctuation.bracket)
(box_object_id left: (box_ident) @variable)
(box_object_id right: (box_ident) @variable)
(box_object_id name: (box_ident) @module)
(box_object_id "/" @punctuation.delimiter)
(box_type "!" @punctuation.special)
(box_type name: (box_ident) @type)
(box_tag (box_ident) @constant)
(box_computed_tag) @comment
(box_field key: (box_ident) @property)
(box_field "=" @operator)
(box_field value: (box_bare_value) @string)
(box_quoted) @string
(box_escape) @string.escape
(box_blob "@" @punctuation.special)

; --- markl ids (shared) ---
(markl_purpose) @property
(markl_format) @type
(markl_id "-" @punctuation.delimiter)
(markl_data) @string.special

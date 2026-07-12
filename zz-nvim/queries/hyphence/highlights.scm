; Highlights for the hyphence metadata block (docs/man.7/hyphence.md).
; Preserves the semantic contract from zz-vim/syntax/dodder-metadata.vim:
;   description -> Title, tag -> Constant, type -> Type, comment -> Comment,
;   blob path base -> Underlined, ext -> Type.

; --- fences ---
(metadata "---" @punctuation.special)

; --- description ( # text ) : Title ---
(description_line "#" @markup.heading.marker)
(description_line text: (description_text) @markup.heading)

; --- tag ( - name ) : Constant ---
(tag_line "-" @punctuation.special)
(tag_line name: (tag_name) @constant)

; --- explicit reference ( < object-id ) ---
(ref_line "<" @punctuation.special)
(ref_line ref: (reference_id) @constant)

; --- type ( ! type-string[@lock] ) : Type ---
(type_line "!" @punctuation.special)
(type_line type: (type_name) @type)
(type_line "@" @punctuation.delimiter)

; --- blob ( @ markl-id | @ file-path ) ---
(blob_line "@" @punctuation.special)
(blob_line ref: (file_path) @string.special.path)

; --- comment ( % text ) : Comment ---
(comment_line "%" @comment)
(comment_line text: (comment_text) @comment)

; --- markl ids (shared: purpose@format-data) ---
(markl_purpose) @property
(markl_format) @type
(markl_id "-" @punctuation.delimiter)
(markl_data) @string.special

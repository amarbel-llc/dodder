; Inject the object body's own language (markdown, toml, ...) into the (body)
; node. The body language is not statically known -- it depends on the object's
; type -- so resolution is delegated to a custom directive registered in
; lua/dodder/injection.lua (#dodder-injection-language!). The directive reads
; the per-buffer resolved language (set asynchronously and cached from
; `dodder show -format type.vim-syntax-type`), falling back to a static table
; keyed on the captured type string, then to markdown. This mirrors the
; shell-out in zz-vim/syntax/dodder-object.vim.

((source_file
   (metadata (type_line type: (type_name) @_type)?)
   (body) @injection.content)
 (#dodder-injection-language! @_type)
 (#set! injection.include-children))

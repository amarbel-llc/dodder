-- Filetype detection for dodder objects, replacing zz-vim/ftdetect.
--
-- Zettel/tag/type/kasten/konfig objects all map to the single `dodder-object`
-- filetype (their bodies differ, but the metadata grammar is shared and body
-- language is resolved per object). Workspace files map to `dodder-workspace`.
--
-- The extensions are the dodder defaults (see
-- go/internal/bravo/file_extensions); they are overlay-configurable, so
-- M.setup accepts an `extensions` list to extend the mapping. The organize
-- filetype (`dodder-organize`) has no stable extension -- organize temp files
-- use the configurable Organize extension (default `md`) -- so, like zz-vim, it
-- is activated programmatically (e.g. `:set ft=dodder-organize`) rather than by
-- an extension rule that would hijack markdown.

local M = {}

local default_object_exts = { "zettel", "tag", "type", "kasten", "konfig" }

function M.setup(opts)
	opts = opts or {}
	local extension = {}
	for _, ext in ipairs(default_object_exts) do
		extension[ext] = "dodder-object"
	end
	for _, ext in ipairs(opts.extensions or {}) do
		extension[ext] = "dodder-object"
	end

	vim.filetype.add({
		extension = extension,
		filename = {
			[".dodder-workspace"] = "dodder-workspace",
			[".zit-workspace"] = "dodder-workspace",
		},
	})
end

return M

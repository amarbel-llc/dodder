-- Body-language injection for hyphence documents.
--
-- The language of a hyphence object's body is not statically known -- it comes
-- from the object's type (its `vim-syntax-type` field). This module mirrors the
-- shell-out in zz-vim/syntax/dodder-object.vim, but non-blocking:
--
--   * a custom tree-sitter directive (#dodder-injection-language!) resolves the
--     body language on every reparse from a cheap per-buffer cache, with a
--     static fallback keyed on the type string so the first paint has no flicker;
--   * an async resolver runs `dodder show -format type.vim-syntax-type` once per
--     buffer, maps the result to a tree-sitter parser, caches it in
--     b:dodder_body_lang, and restarts highlighting so injections re-run.

local M = {}

-- Map a resolved value to a tree-sitter parser name. `vim-syntax-type` values
-- are real vim filetypes (sh, javascript, dot, ...), so neovim's own
-- filetype->parser mapping does most of the work; this table covers the few
-- where the parser name differs from the filetype in a way get_lang misses.
local ts_lang_override = {
	svg = "xml",
	cfg = "toml",
}

--- Resolve a dodder type string to a tree-sitter language for the fast path.
-- Only versioned config/type blobs (toml-...-vN, ...-config-...) get a
-- non-default guess; everything else (md, task, chore, txt, ...) renders as
-- markdown until the async resolver corrects it.
function M.static_lang(type_string)
	if type_string and (type_string:match("^toml%-") or type_string:match("config")) then
		return "toml"
	end
	return "markdown"
end

--- Map a vim filetype / vim-syntax-type value to an available ts parser name.
function M.ts_lang_for(vim_syntax_type)
	if not vim_syntax_type or vim_syntax_type == "" then
		return "markdown"
	end
	local lang = ts_lang_override[vim_syntax_type]
		or vim.treesitter.language.get_lang(vim_syntax_type)
		or vim_syntax_type
	return lang
end

--- Is a parser for `lang` loadable?
local function has_parser(lang)
	return pcall(vim.treesitter.language.add, lang)
end

--- Cache the resolved body language and restart highlighting if it changed.
function M.set_lang(bufnr, lang)
	if not (lang and has_parser(lang)) then
		lang = "markdown"
	end
	if vim.b[bufnr].dodder_body_lang == lang then
		return
	end
	vim.b[bufnr].dodder_body_lang = lang
	-- Restart the highlighter so injection ranges are recomputed with the new
	-- language. Cheaper and more reliable than poking the injection query cache.
	if vim.api.nvim_buf_is_valid(bufnr) then
		pcall(vim.treesitter.stop, bufnr)
		pcall(vim.treesitter.start, bufnr, "hyphence")
	end
end

--- Resolve the body language for a buffer asynchronously via the dodder CLI.
function M.resolve_async(bufnr)
	bufnr = bufnr or vim.api.nvim_get_current_buf()
	local name = vim.api.nvim_buf_get_name(bufnr)
	if name == "" then
		return
	end

	-- Config (.konfig) and workspace bodies are always TOML (matching
	-- zz-vim/ftplugin/dodder-konfig.vim and syntax/dodder-workspace.vim).
	local ext = name:match("%.([^.\\/]+)$")
	if vim.bo[bufnr].filetype == "dodder-workspace" or ext == "konfig" then
		M.set_lang(bufnr, "toml")
		return
	end

	local bin = require("dodder").bin()
	if vim.fn.executable(bin) ~= 1 then
		-- No dodder on PATH: keep the static fallback (markdown / forced toml).
		return
	end
	local cmd = { bin, "show", "-quiet", "-format", "type.vim-syntax-type", name }

	local function apply(code, stdout)
		vim.schedule(function()
			-- Guard against buffer-number reuse: if bufnr was wiped and
			-- reassigned to a different file while this request was in flight,
			-- nvim_buf_is_valid alone would still say "valid" for the new file.
			if not vim.api.nvim_buf_is_valid(bufnr) or vim.api.nvim_buf_get_name(bufnr) ~= name then
				return
			end
			local out = (stdout or ""):gsub("%s+$", "")
			if code == 0 and out ~= "" then
				M.set_lang(bufnr, M.ts_lang_for(out))
			else
				-- No vim-syntax-type set, or the CLI errored (e.g. object not in a
				-- workspace): fall back to markdown, as zz-vim does.
				M.set_lang(bufnr, "markdown")
			end
		end)
	end

	if vim.system then
		vim.system(cmd, { text = true }, function(res)
			apply(res.code, res.stdout)
		end)
	else
		-- neovim < 0.10 has no vim.system; use jobstart to stay non-blocking.
		local chunks = {}
		vim.fn.jobstart(cmd, {
			stdout_buffered = true,
			on_stdout = function(_, data)
				if data then
					vim.list_extend(chunks, data)
				end
			end,
			on_exit = function(_, code)
				apply(code, table.concat(chunks, "\n"))
			end,
		})
	end
end

--- The directive read on every reparse: set injection.language from the cache,
--- else the static fallback keyed on the captured type string, else markdown.
local function directive(match, _, bufnr, pred, metadata)
	local lang = vim.b[bufnr].dodder_body_lang
	if not lang then
		local capture_id = pred[2]
		local type_string
		if capture_id then
			local node = match[capture_id]
			if type(node) == "table" then
				node = node[1]
			end
			if node then
				type_string = vim.treesitter.get_node_text(node, bufnr)
			end
		end
		lang = M.static_lang(type_string)
	end
	metadata["injection.language"] = lang
end

local did_setup = false

--- Register the injection directive and body-language aliases (idempotent).
function M.setup()
	if did_setup then
		return
	end
	did_setup = true

	vim.treesitter.query.add_directive("dodder-injection-language!", directive, { force = true, all = false })

	-- Belt-and-suspenders: make plain filetype/language references resolve too.
	vim.treesitter.language.register("markdown", { "md", "task" })
	vim.treesitter.language.register("bash", { "sh" })
	vim.treesitter.language.register("xml", { "svg" })
end

return M

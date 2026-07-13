-- dodder.nvim: tree-sitter highlighting and editor integration for dodder.
--
-- Replaces zz-vim's syntax highlighting with tree-sitter grammars (hyphence,
-- doddish, dodder_organize) plus per-object body-language injection, and ports
-- zz-vim's behavioral ftplugin features (format-on-=, gf navigation, object
-- action/copy/preview menus, organize folding).
--
-- Usage: require('dodder').setup()  -- or just load the plugin (plugin/dodder.lua
-- calls setup with defaults). Options:
--   { bin = 'dodder', extensions = { 'myext' }, ftplugin = true }

local M = {}

local defaults = {
	-- dodder binary; also honors vim.g.dodder_bin and $BIN_DODDER.
	bin = nil,
	-- extra object file extensions beyond the dodder defaults.
	extensions = {},
	-- port zz-vim's behavioral ftplugin (keymaps, equalprg, folding).
	ftplugin = true,
}

M.config = defaults

-- Filetype -> tree-sitter parser. dodder-object and dodder-workspace both use
-- the hyphence parser (workspace bodies are forced to TOML by the resolver).
local ft_to_lang = {
	["dodder-object"] = "hyphence",
	["dodder-workspace"] = "hyphence",
	["dodder-organize"] = "dodder_organize",
}

local resolver_exts = { "*.zettel", "*.tag", "*.type", "*.kasten", "*.konfig" }

--- The dodder binary to invoke: vim.g.dodder_bin, then $BIN_DODDER, then 'dodder'.
function M.bin()
	return vim.g.dodder_bin or vim.env.BIN_DODDER or "dodder"
end

function M.setup(opts)
	M.config = vim.tbl_deep_extend("force", defaults, opts or {})
	if M.config.bin then
		vim.g.dodder_bin = M.config.bin
	end

	require("dodder.filetype").setup({ extensions = M.config.extensions })

	for ft, lang in pairs(ft_to_lang) do
		vim.treesitter.language.register(lang, ft)
	end

	require("dodder.injection").setup()

	local group = vim.api.nvim_create_augroup("dodder", { clear = true })

	-- Start tree-sitter highlighting and run the ftplugin for dodder buffers.
	vim.api.nvim_create_autocmd("FileType", {
		group = group,
		pattern = { "dodder-object", "dodder-workspace", "dodder-organize" },
		callback = function(ev)
			pcall(vim.treesitter.start, ev.buf)
			if ev.match == "dodder-object" or ev.match == "dodder-workspace" then
				require("dodder.injection").resolve_async(ev.buf)
			end
			if M.config.ftplugin then
				require("dodder.ftplugin").on_filetype(ev)
			end
		end,
	})

	-- Re-resolve the body language when the object's type may have changed.
	vim.api.nvim_create_autocmd("BufWritePost", {
		group = group,
		pattern = resolver_exts,
		callback = function(ev)
			require("dodder.injection").resolve_async(ev.buf)
		end,
	})
end

return M

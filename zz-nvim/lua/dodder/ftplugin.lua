-- Behavioral ftplugin, ported from zz-vim/ftplugin/dodder-object.vim and
-- dodder-organize.vim to lua. Buffer-local options, `gf` navigation, the
-- object action/copy/preview menus, and organize folding.

local M = {}

local function bin()
	return require("dodder").bin()
end

--- Run `dodder` and return trimmed stdout (or '' on error).
local function run(args)
	local out = vim.fn.system(vim.list_extend({ bin() }, args))
	if vim.v.shell_error ~= 0 then
		return ""
	end
	return vim.trim(out)
end

--- The object id for the current buffer (its file path, as zz-vim used).
local function object_id()
	return vim.fn.expand("%")
end

-- `dodder show -format type.<what>` returns lines of "<id><whitespace><desc>".
-- Return a list of { id = first token, label = whole line }.
local function id_list(what)
	local id = object_id()
	local lines = vim.fn.systemlist({ bin(), "show", "-format", "type." .. what, id })
	if vim.v.shell_error ~= 0 then
		return {}
	end
	table.sort(lines)
	local items = {}
	for _, line in ipairs(lines) do
		line = vim.trim(line)
		if line ~= "" then
			items[#items + 1] = { id = line:match("^(%S+)") or line, label = line }
		end
	end
	return items
end

--- Notify and return false if `exe` isn't on PATH.
local function require_executable(exe, message)
	if vim.fn.executable(exe) == 1 then
		return true
	end
	vim.notify("dodder: " .. message, vim.log.levels.WARN)
	return false
end

local function select_item(what, prompt, on_choice)
	local items = id_list(what)
	if #items == 0 then
		vim.notify("dodder: no " .. prompt .. " available for this object", vim.log.levels.INFO)
		return
	end
	if #items == 1 then
		on_choice(items[1])
		return
	end
	vim.ui.select(items, {
		prompt = prompt,
		format_item = function(item)
			return item.label
		end,
	}, function(item)
		if item then
			on_choice(item)
		end
	end)
end

--- gf on a zettel id: expand it, check it out if needed, open in a new tab.
function M.gf_zettel()
	local cfile = vim.fn.expand("<cfile>")
	local expanded = run({ "expand-zettel-id", cfile })
	if expanded == "" then
		vim.notify("dodder: could not expand " .. cfile, vim.log.levels.WARN)
		return
	end
	local file = expanded .. ".zettel"
	if vim.fn.filereadable(file) == 0 then
		run({ "checkout", "-mode", "both", expanded })
		if vim.v.shell_error ~= 0 then
			vim.notify("dodder: checkout failed for " .. expanded, vim.log.levels.ERROR)
			return
		end
	end
	vim.cmd.tabedit(vim.fn.fnameescape(file))
end

--- Run a type-specific action on the object.
function M.action()
	select_item("action-names", "Run a type-specific action", function(item)
		vim.fn.system({ bin(), "exec-action", "-action", item.id, object_id() })
		vim.cmd.checktime()
	end)
end

--- Preview the object rendered through one of its type's formatters.
function M.preview()
	if not require_executable("qlmanage", "preview needs `qlmanage` (macOS Quick Look)") then
		return
	end
	select_item("formatters", "Preview format", function(item)
		-- format id and file extension are the two columns of the line.
		local fmt, ext = item.label:match("^(%S+)%s+(%S+)")
		fmt = fmt or item.id
		local tmp = vim.fn.tempname() .. (ext and ("." .. ext) or "")
		vim.fn.system(
			("%s format-object -mode blob %s %s > %s"):format(
				vim.fn.shellescape(bin()),
				vim.fn.shellescape(object_id()),
				vim.fn.shellescape(fmt),
				vim.fn.shellescape(tmp)
			)
		)
		vim.fn.jobstart({ "qlmanage", "-p", tmp }, { detach = true })
	end)
end

--- Copy the object's blob(s) to the clipboard via a UTI group (macOS `tacky`).
function M.copy()
	if not require_executable("tacky", "copy needs `tacky`") then
		return
	end
	select_item("formatter-uti-groups", "Copy format", function(item)
		vim.fn.system({ "tacky", "copy", "-i", item.id, object_id() })
	end)
end

--- foldexpr for organize buffers: heading depth from leading '#'.
function M.organize_foldexpr()
	local line = vim.fn.getline(vim.v.lnum)
	local hashes = line:match("^(#+)%s")
	if hashes then
		return ">" .. #hashes
	end
	return "="
end

--- foldtext for organize buffers: "+-- N lines: <heading>".
function M.organize_foldtext()
	local line = vim.fn.getline(vim.v.foldstart)
	local n = vim.v.foldend - vim.v.foldstart
	return ("%s %d lines: %s"):format(vim.v.folddashes, n, line)
end

local function setup_object(buf)
	vim.bo[buf].equalprg = bin() .. " format-object -mode both %"
	vim.bo[buf].commentstring = "<!--%s-->"
	vim.bo[buf].comments = "fb:*,fb:-,fb:+,n:>"

	local opts = { buffer = buf, silent = true }
	vim.keymap.set("n", "gf", M.gf_zettel, opts)
	vim.keymap.set("n", "<localleader>z", M.action, opts)
	vim.keymap.set("n", "<localleader>c", M.copy, opts)
	vim.keymap.set("n", "<localleader>p", M.preview, opts)
end

local function setup_organize(buf, win)
	vim.bo[buf].equalprg = bin() .. " format-organize -metadata-header %"
	vim.bo[buf].commentstring = "# %s"

	-- FileType events don't carry a window, and the buffer may not be shown
	-- in the current window (e.g. loaded in the background) -- apply the
	-- window-local fold options to every window actually displaying this
	-- buffer, falling back to the current window if none do yet.
	local wins = win and { win } or vim.fn.win_findbuf(buf)
	if #wins == 0 then
		wins = { vim.api.nvim_get_current_win() }
	end
	for _, w in ipairs(wins) do
		vim.wo[w].list = true
		vim.wo[w].listchars = "tab:  ,trail:·,nbsp:·"
		vim.wo[w].foldmethod = "expr"
		vim.wo[w].foldexpr = 'v:lua.require("dodder.ftplugin").organize_foldexpr()'
		vim.wo[w].foldtext = 'v:lua.require("dodder.ftplugin").organize_foldtext()'
	end
	vim.keymap.set("n", "gf", M.gf_zettel, { buffer = buf, silent = true })
end

--- Dispatch buffer-local setup by filetype (called from the FileType autocmd).
function M.on_filetype(ev)
	vim.b[ev.buf].maplocalleader = "-"
	if ev.match == "dodder-object" then
		setup_object(ev.buf)
	elseif ev.match == "dodder-workspace" then
		vim.bo[ev.buf].commentstring = "# %s"
	elseif ev.match == "dodder-organize" then
		setup_organize(ev.buf)
	end
end

return M

-- :checkhealth dodder
--
-- Verifies the dodder binary resolves, the three tree-sitter parsers load, and
-- the shipped queries parse.

local M = {}

local function bin()
  return vim.g.dodder_bin or vim.env.BIN_DODDER or 'dodder'
end

local parsers = { 'hyphence', 'doddish', 'dodder_organize' }

function M.check()
  local health = vim.health or require('health')
  health.start('dodder.nvim')

  -- binary
  local exe = bin()
  if vim.fn.executable(exe) == 1 then
    local ok, out = pcall(vim.fn.system, { exe, '-version' })
    health.ok(('dodder binary: %s%s'):format(
      exe,
      (ok and out and out ~= '') and (' (' .. vim.trim(out) .. ')') or ''
    ))
  else
    health.warn(
      ('dodder binary %q not found on PATH'):format(exe),
      { 'set vim.g.dodder_bin or $BIN_DODDER', 'body-language resolution falls back to markdown' }
    )
  end

  -- parsers
  for _, lang in ipairs(parsers) do
    if pcall(vim.treesitter.language.add, lang) then
      health.ok(('parser installed: %s'):format(lang))
    else
      health.error(('parser missing: %s'):format(lang), {
        'build parsers via the nix `dodder-nvim` package, or',
        'add the grammar dirs to nvim-treesitter and :TSInstall ' .. lang,
      })
    end
  end

  -- queries
  for _, lang in ipairs({ 'hyphence', 'doddish', 'dodder_organize' }) do
    local ok = pcall(vim.treesitter.query.get, lang, 'highlights')
    if ok then
      health.ok(('highlights query loads: %s'):format(lang))
    else
      health.warn(('highlights query not found for %s'):format(lang))
    end
  end

  local ok_inj = pcall(vim.treesitter.query.get, 'hyphence', 'injections')
  if ok_inj then
    health.ok('hyphence injections query loads')
  else
    health.warn('hyphence injections query not found')
  end
end

return M

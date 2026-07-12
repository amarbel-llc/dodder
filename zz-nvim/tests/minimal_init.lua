-- Minimal init for testing dodder.nvim in isolation:
--   nvim --clean -u zz-nvim/tests/minimal_init.lua path/to/file.zettel
-- Point ZZ_NVIM at the plugin root, or run from the dodder repo root.
local root = vim.env.ZZ_NVIM or (vim.fn.getcwd() .. '/zz-nvim')
vim.opt.runtimepath:prepend(root)
vim.g.mapleader = ' '
require('dodder').setup()

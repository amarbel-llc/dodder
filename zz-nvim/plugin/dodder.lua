-- Auto-load entry point: set up dodder.nvim with defaults unless the user has
-- opted out (vim.g.dodder_no_default_setup = true to call require('dodder').setup
-- manually with custom options).
if vim.g.loaded_dodder_nvim then
	return
end
vim.g.loaded_dodder_nvim = true

if not vim.g.dodder_no_default_setup then
	require("dodder").setup()
end

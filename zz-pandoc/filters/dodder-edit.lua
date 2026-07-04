package.path = package.path .. string.format(";%s/.local/share/pandoc/filters/?.lua", os.getenv("HOME"))

local pandoc = require("pandoc")
local common = require("dodder-common")

-- Image = common.try_to_replace_image_with_new_or_added_object_link

function CodeBlock(el)
	local classes = el.classes

	if #classes < 1 then
		return nil
	end

	local type = classes[1]

	if type:find("^!") == nil then
		return nil
	end

	local data = pandoc.pipe("dodder", { "format-object", "-stdin", type }, el.text)

	el.text = data

	return el
end

function Link(el)
	common.unescape_if_sku(el, "target")
	return el
end

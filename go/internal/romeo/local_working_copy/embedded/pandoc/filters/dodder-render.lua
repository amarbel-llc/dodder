local blob_tree = os.getenv("DODDER_BLOB_TREE") or ""
package.path = package.path .. string.format(";%s/filters/?.lua", blob_tree)

local pandoc = require("pandoc")
local common = require("dodder-common")

-- if common.IsBinary then
--   function Image(img)
--     return common.replace_object_image_with_image_if_necessary(img)
--   end
-- else
--   function Figure(el)
--     local content = el.content

--     if #content ~= 1 then
--       return el
--     end

--     if content[1].t ~= "Plain" then
--       return el
--     end

--     local plain = content[1]
--     local image = plain.content[1]
--     local replacement = common.replace_object_image_with_text_if_necessary(image)

--     if replacement == nil then
--       return el
--     else
--       return replacement
--     end
--   end
-- end

-- TODO add code formatter
function CodeBlock(el)
	local classes = el.classes

	if #classes < 1 then
		return nil
	end

	local type = classes[1]

	if type:find("^!") == nil then
		return nil
	end

	local format = "text"

	if common.IsBinary then
		format = "png"
	end

	format = "png"
	local data = pandoc.pipe("dodder", { "format-object", "-stdin", format, type }, el.text)

	local id = pandoc.utils.sha1(el.text)
	local fname = id .. ".png"

	local file = io.open(fname, "wb")
	file:write(data)
	file:close()

	-- pandoc.mediabag.insert(fname, "image/png", data)
	return pandoc.Image("", fname)
end

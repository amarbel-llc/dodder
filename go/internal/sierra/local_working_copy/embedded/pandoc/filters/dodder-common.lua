local pandoc = require("pandoc")
local url = require("url")
local P = {}

---@diagnostic disable-next-line: undefined-global
function P.is_binary()
  return FORMAT:find("^markdown") == nil
end

function P.format_object_image(imgSrc, format)
  local objectID = url.unescape(imgSrc)
  return pandoc.pipe("dodder", { "format-object", objectID, format }, ""), objectID
end

local mimeTypeMapping = {
  ["image/jpeg"] = "jpg",
  ["image/png"] = "png",
  ["image/gif"] = "gif",
}

local function try_to_replace_src_with_new_or_added_object(
  urlOrFileEscaped,
  captionInlines
  )
  -- TODO [sta/sud @765e !task project-2021-zit-features-user_story-editing zz-inbox] add support for organize-like new object declarations in blobs via …
  if P.is_sku(urlOrFileEscaped) then
    return urlOrFileEscaped
  end

  local tipe, data
  local description = ""

  if urlOrFileEscaped:find("^http") ~= nil then
    local mime

    mime, data = pandoc.mediabag.fetch(urlOrFileEscaped)

    -- TODO transform mime into type via config
    tipe = mimeTypeMapping[mime]
  else
    -- TODO modify this to do `add` instead of `new`
    local path = url.unescape(urlOrFileEscaped)
    local f = io.open(path, "rb")

    if f == nil then
      return urlOrFileEscaped
    end

    data = f:read("*all")
    tipe = string.match(path, "%.(%w+)$")
    description = path
  end

  if tipe == nil then
    tipe = ""
  end

  if captionInlines ~= nil then
    local captionString = pandoc.utils.stringify(captionInlines)

    if captionString ~= "" then
      description = captionString
    end
  end

  if description == "" then
    description = urlOrFileEscaped
  end

  -- TODO add mediabag tags

  local args = {
    "new",
    "-abbreviate-zettel-ids=false",
    "-abbreviate-shas=false",
    "-print-time=false",
    "-print-bestandsaufnahme=false",
    "-edit=false",
    -- TODO read dry run from dodder config
    "-dry-run",
    "-type", tipe,
    "-description", description,
    "-",
  }

  local result = pandoc.pipe("dodder", args, data)

  -- strip newline
  urlOrFileEscaped = string.sub(result, 1, #result - 1)

  return urlOrFileEscaped
end

function P.try_to_replace_image_with_new_or_added_object_link(img)
  P.unescape_if_sku(img, "src")

  -- TODO add flag for determining if this should be added to dodder
  if false then
    return img
  end

  img.src = try_to_replace_src_with_new_or_added_object(img.src, img.caption)

  return img
end

-- [fib/chil @b7a8 !task project-2021-zit-bugs zz-inbox] modify pandoc filters to handle non-objects in images
function P.replace_object_image_with_text_if_necessary(img)
  if img.src == nil then
    return nil
  end

  local format = "text"

  local data, _ = P.format_object_image(img.src, format)

  return pandoc.RawInline("markdown", data)
end

-- [fib/chil @b7a8 !task project-2021-zit-bugs zz-inbox] modify pandoc filters to handle non-objects in images
function P.replace_object_image_with_image_if_necessary(img)
  local src = img.src

  if src == nil then
    return nil
  end

  local format, data, fname, mime

  if src:find("^dodder://blobs/") == nil then
    format = "png"
    mime = "image/png"

    local objectId
    data, objectId = P.format_object_image(img.src, format)

    local id = pandoc.utils.sha1(objectId)
    fname = string.format("%s.%s", id, format)
  else
    local shaStart, shaEnd = string.find(src, "[^/]+", 13)

    if shaStart == nil then
      return img
    end

    local sha = string.sub(src, shaStart, shaEnd)

    local mimeStart, mimeEnd = string.find(src, "[^/]+", shaEnd + 1)

    if mimeStart == nil then
      return img
    end

    mime = string.sub(src, mimeStart, mimeEnd)
    data = pandoc.pipe("dodder", { "cat-blob", sha }, "")
  end

  pandoc.mediabag.insert(fname, mime, data)

  -- TODO try to extract caption from sku if caption is empty

  return pandoc.Image(img.caption, fname)
end

-- [fib/chil @b7a8 !task project-2021-zit-bugs zz-inbox] modify pandoc filters to handle non-objects in images
function P.is_sku(str)
  local found = string.find(str, "^%s*%[")
  return found ~= nil
end

function P.unescape_if_sku(table, key)
  local el = table[key]
  local unescaped = url.unescape(el)

  if not P.is_sku(unescaped) then
    return
  end

  table[key] = unescaped
end

return P

function Writer(doc, opts)
  local seen = {}
  local refs = {}

  doc:walk({
    Link = function(link)
      if link.classes:includes("wikilink") then
        local target = link.target
        if not seen[target] then
          seen[target] = true
          refs[#refs + 1] = target
        end
      end
    end,
  })

  return table.concat(refs, "\n") .. (#refs > 0 and "\n" or "")
end

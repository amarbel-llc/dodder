-- actionable-common: shared commit hook for the built-in actionable types
-- (!task/!chore/!habit). Blob-backed and loaded via `require` through the
-- dodder object graph (FDR-0000): the type objects carry this as a blob
-- reference, preloaded into the hook VM by name (see oscar/store).
--
-- on_commit_fields runs after the commit pipeline projects fields into
-- kinder.Fields. Behavior (field model):
--   * status=="cancelled": archive (zz-archive tag) for every actionable type,
--     and stamp today into an empty `due` (completed-on).
--   * status=="done" on !task: archive + stamp today into an empty `due`.
--   * status=="done" on a recurring type (!chore/!habit) with non-empty
--     recurrence: advance `due` by the recurrence (host dodder_advance_date)
--     and reset status to "todo".
-- The "zz-archive" literal MUST match type_blobs.ArchiveTag.
local P = {}

local function today()
	return dodder_today()
end

local function archive(kinder)
	kinder.Etiketten["zz-archive"] = true
	local f = kinder.Fields
	if f.due == nil or f.due == "" then
		f.due = today()
	end
end

function P.on_commit_fields(kinder, mutter)
	local f = kinder.Fields
	if not f then
		return
	end
	local status = f.status
	if status == "cancelled" then
		archive(kinder)
	elseif status == "done" then
		if kinder.Typ == "!task" then
			archive(kinder)
		elseif f.recurrence ~= nil and f.recurrence ~= "" then
			if f.due ~= nil and f.due ~= "" then
				f.due = dodder_advance_date(f.due, f.recurrence)
			end
			f.status = "todo"
		end
	end
end

P.hooks = {
	on_commit_fields = P.on_commit_fields,
}

return P

-- actionable-common: shared commit hook for the built-in actionable types
-- (!task/!chore/!habit). Blob-backed and loaded via `require` through the
-- dodder object graph (FDR-0000): the type objects carry this as a blob
-- reference, preloaded into the hook VM by name (see oscar/store).
--
-- Written against the English-keyed hook tables (child.Type, child.Fields):
-- the built-in actionable types are !toml-type-v3, whose hooks receive that
-- projection (dodder#407). Repos created earlier keep their own committed
-- copy of this module, written against the German keys, on their v2 types.
--
-- on_commit_fields runs after the commit pipeline projects fields into
-- child.Fields. Behavior (field model):
--   * status=="cancelled": stamp today into an empty `due` (completed-on).
--   * status=="done" on !task: stamp today into an empty `due`.
--   * status=="done" on a recurring type (!chore/!habit) with non-empty
--     recurrence: advance `due` by the recurrence (host dodder_advance_date)
--     and reset status to "todo".
-- Dormancy is NOT this hook's job: the types declare their terminal status
-- values on the status field and dodder makes such objects dormant directly
-- (FDR 0025).
local P = {}

local function today()
	return dodder_today()
end

local function stamp_completed_on(f)
	if f.due == nil or f.due == "" then
		f.due = today()
	end
end

function P.on_commit_fields(child, mother)
	local f = child.Fields
	if not f then
		return
	end
	local status = f.status
	if status == "cancelled" then
		stamp_completed_on(f)
	elseif status == "done" then
		if child.Type == "!task" then
			stamp_completed_on(f)
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

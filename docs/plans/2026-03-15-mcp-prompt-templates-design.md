# MCP Prompt Templates for Progressive Disclosure

**Date**: 2026-03-15
**Status**: proposed

## Problem

Agents using the dodder MCP struggle with three failure modes:

1. **Wrong tool selection** — calling `dodder_query` or `dodder_show` when
   `tag_query`/`type_query` should be the entry point, producing unbounded
   result sets
2. **Missing resource follow-up** — getting tool results with `resource-uri`
   fields but not reading them, stopping at the first result
3. **Can't compose multi-step workflows** — understanding individual tools but
   failing to chain discovery → filter → drill-down → content

The `mcpInstructions` string teaches protocol mechanics but doesn't provide
executable recipes. Agents need step-by-step workflows they can follow
mechanically.

## Approach: Workflow Prompts

Register goal-oriented MCP prompt templates via `server.NewPromptRegistry()`.
Each prompt returns a single `user`-role message containing a numbered recipe
with exact tool calls, argument values, and "what to look for" annotations.

Prompts are static string templates with argument substitution — no bridge
calls, no CLI invocations at render time.

## Prompt Catalog

| Prompt | Arguments | Workflow |
|--------|-----------|----------|
| `summarize-projects` | none | tag_query → filter meta-tags for "active" → facets drill-down |
| `find-tasks` | `priority` (opt), `tag` (opt) | dodder_query with AND-combined filters + limit |
| `read-object` | `object_id` (req) | show → blob format discovery → blob render |
| `explore-type` | `type` (req) | type_query → type detail → facets → sample objects |
| `explore-tag` | `tag` (req) | tag_query → tag detail → facets → objects |

## Message Structure

Each prompt returns markdown with numbered steps:

- Explicit tool names and arguments (no ambiguity)
- "What to look for" annotations (e.g., "filter where tags contains active")
- Resource URI templates with `<placeholder>` explained in context
- Agents parse this naturally as instructions

## Files Modified

1. **`go/internal/tango/mcp_dodder/prompts.go`** (new) — prompt metadata,
   renderer functions, `registerPrompts` function
2. **`go/internal/tango/mcp_dodder/server.go`** — create `PromptRegistry`,
   call `registerPrompts`, add `Prompts` to `server.Options`

## Rollback

Additive change. Remove `Prompts` field from `server.Options` to restore
previous behavior. No existing tools, resources, or instructions change.

# Claude Skills Plugin Design

## Summary

Add a purse-first skill-only plugin to dodder, shipping three Claude Code
skills: development (codebase contribution), onboarding (getting started), and
usage (day-to-day workflows). Later to be registered with the purse-first
marketplace.

## Plugin Structure

```
.claude-plugin/
  plugin.json

skills/
  dodder-development/
    SKILL.md
    references/
      nato-hierarchy.md
      pool-management.md
      testing-workflow.md
  dodder-onboarding/
    SKILL.md
    references/
      concepts.md
  dodder-usage/
    SKILL.md
    references/
      commands.md
      querying.md
```

### plugin.json

Skill-only plugin, no MCP servers:

```json
{
  "name": "dodder",
  "skills": [
    "./skills/dodder-development",
    "./skills/dodder-onboarding",
    "./skills/dodder-usage"
  ]
}
```

## Skill 1: dodder-development

**Audience:** Contributors working on the dodder Go codebase.

**Triggers:** "work on dodder", "add a new command", "add a type", "register a
coder", "fix dodder code", "add a NATO module", "repool", "pool management",
"dodder architecture"

**SKILL.md (~1,800 words):**
- Project overview (Go, distributed zettelkasten, dodder/der binaries)
- Build/test cycle (just build, just test, just test-bats, fixture workflow)
- NATO phonetic module hierarchy rules and dependency direction
- Critical rules: never dereference sku.Transacted pointers, use ResetWith,
  pool management with GetWithRepool
- Repool analyzer usage
- Codebase navigation (cmd/ for entry points, src/ for NATO layers)
- Pointers to CLAUDE.md and MIGRATION_LEARNINGS.md

**References:**
- `nato-hierarchy.md` — Full layer breakdown with contents, dependency rules,
  correct vs. incorrect import examples
- `pool-management.md` — Repool semantics, the static analyzer, common
  mistakes, debugging pool leaks
- `testing-workflow.md` — BATS fixture lifecycle, writing new integration tests,
  test-bats-tags, fixture regeneration

## Skill 2: dodder-onboarding

**Audience:** New users encountering dodder for the first time.

**Triggers:** "get started with dodder", "set up dodder", "install dodder",
"new to dodder", "what is dodder", "dodder tutorial", "learn dodder"

**SKILL.md (~1,500 words):**
- What dodder is (distributed zettelkasten, content-addressable, like Git for
  notes)
- Core concepts: zettels, object IDs (two-part identifiers), tags, types
- Installation (nix-based, devshell)
- First steps: dodder init, creating first zettel, checking status
- Mental model: flat hierarchy, everything versioned, no directories
- Next steps: points to the dodder-usage skill

**References:**
- `concepts.md` — Deeper explanation of object model, ID system, type system,
  versioning, content-addressable storage

## Skill 3: dodder-usage

**Audience:** Existing dodder users performing day-to-day operations.

**Triggers:** "create a zettel", "organize zettels", "dodder checkout",
"dodder push", "dodder pull", "query dodder", "find zettels", "sync dodder",
"tag a zettel", "edit a zettel", "dodder commands", "der"

**SKILL.md (~2,000 words):**
- dodder and der are the same tool (der is a shortname)
- madder excluded (low-level, separate skill later)
- Common workflows:
  - Creating and editing zettels
  - Checking out working copies
  - Organizing (moving, tagging, typing)
  - Querying and filtering
  - Push/pull with remotes
- Working copy vs. store distinction
- Tags and types as first-class concepts

**References:**
- `commands.md` — Command reference organized by workflow with examples
- `querying.md` — Query syntax, filtering, combining queries, examples

## Future Work

- `dodder-madder` skill for low-level blob/repo operations
- Register plugin with purse-first marketplace
- Nix build integration (copy skills to $out/share/purse-first/dodder/)

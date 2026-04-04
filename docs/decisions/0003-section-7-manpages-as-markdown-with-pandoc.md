---
date: 2026-04-04
status: accepted
---

# Section 7 Manpages as Markdown with Pandoc Conversion

## Context and Problem Statement

Issue #90 adds section 7 manpages (`doddish(7)`, `markl-id(7)`) to document
dodder's query language and identifier format. The existing manpage
infrastructure generates section 1 (command) pages from Go command metadata via
`dodder-gen_man`. Section 7 pages are authored prose, not generated from code.
How should section 7 content be authored, stored, and converted to roff?

## Decision Drivers

- Section 7 pages are hand-written reference documentation, not derived from
  code structure
- RFC specifications exist in `docs/rfcs/` but are too verbose for manpages
  (normative language, security considerations, test vectors, compatibility
  notes)
- The nix build already has pandoc available (used by lux filetype config for
  markdown formatting)
- Manpage content should be editable without recompiling Go binaries

## Considered Options

1.  Embedded Go string constants in a new gen_man binary
2.  Convert RFCs directly to manpages (symlinks or aliases)
3.  Hand-written markdown in `docs/man.7/` with pandoc conversion

## Decision Outcome

Chosen option: **hand-written markdown with pandoc conversion**, because it
keeps documentation as documentation, avoids coupling prose to Go compilation,
and produces appropriately condensed manpages distinct from the verbose RFC
specs.

### Implementation

- Source files live in `docs/man.7/*.md` (one file per manpage)
- The nix build converts each `.md` to roff via `pandoc -s -t man`
- Output goes to `$out/share/man/man7/`
- No changes to the existing `golf/man/` Go package or gen_man binaries

### Relationship to RFCs

The RFCs in `docs/rfcs/` are the authoritative specifications for implementors.
The manpages are condensed quick-references for users --- analogous to
`gitrevisions(7)` vs. the full git documentation. The two cross-reference each
other but serve different audiences and are maintained independently.

## Pros and Cons of the Options

### Embedded Go strings

- Good, because it reuses the existing gen_man pattern
- Bad, because editing manpage content requires recompiling the generator
- Bad, because it puts documentation in Go source instead of a documentation
  directory
- Bad, because it creates a second source of truth alongside the RFCs

### Convert RFCs directly

- Good, because it avoids maintaining two documents
- Bad, because RFCs contain material inappropriate for manpages (YAML front
  matter, requirements language sections, security considerations, test vectors,
  compatibility notes)
- Bad, because a manpage should be a condensed reference, not a full spec

### Hand-written markdown with pandoc

- Good, because content is editable without recompilation
- Good, because markdown is a natural format for authored prose
- Good, because pandoc's man output handles headings, code blocks, bold/italic,
  and tables correctly
- Good, because it cleanly separates the spec (RFC) from the reference (manpage)
- Neutral, because it adds a pandoc build dependency (already available)

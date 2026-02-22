# Doddish Parsing Overlap Analysis

Analysis of common and redundant parsing across `charlie/doddish`,
`kilo/box_format`, `november/queries`, and `oscar/organize_text`.

## Context

Four packages consume doddish token sequences, each with their own scan loop and
sequence interpretation:

- **charlie/doddish** — scanner, tokenizer, Seq type, token matchers
- **kilo/box_format** — reads `[objectId metadata...]` boxes into `*sku.Transacted`
- **november/queries** — parses query expressions into filter/match trees
- **oscar/organize_text** — hierarchical heading+object text format (delegates
  object parsing to box_format)

Both box_format (`read.go:183`) and queries (`build_state.go:223`) have the same
TODO: "convert this into a decision tree based on token type sequences instead of
a switch."

## Identified Opportunities

### A. Shared Seq Dispatch / Decision Tree — REJECTED

**Idea:** Extract the `for scanner.Scan() { switch on seq patterns }` loop into
a reusable `DecisionTree` type in `charlie/doddish` with registered pattern →
handler rules.

**Why rejected:**
- The abstraction doesn't reduce complexity — it moves the switch into a slice of
  closures with identical line count.
- Constructing the rule slice and closures on every call adds allocation overhead.
  box_format's `ReadStringFormat` is called for every object read; allocation cost
  matters here.
- Handlers need control flow signals (break, continue, unscan) back to the loop,
  adding a `LoopAction` return that's noise for 80% of handlers that just proceed.
- The queries consumer has intertwined mutable state (negation/exact flags,
  grouping stack) that doesn't decompose into small independent rules.

### B. Unified ObjectId-from-Seq Parsing — LOW VALUE

**Idea:** Extract the shared ObjectId parsing (SetWithSeq + external fallbacks +
genre inference from `.suffix`) into a reusable function.

**Analysis:**
- The shared core is already factored: `ids.ObjectId.SetWithSeq(seq)` in
  `echo/ids/object_id3.go`.
- box_format has a cascade of external ID fallbacks (literal, absolute path,
  genre-suffixed, plain external) that queries doesn't need.
- queries strips embedded `.genre` as a sigil (`build_state.go:284-310`) which is
  a different operation than box_format's genre inference for ObjectId
  (`read.go:136-160`).
- The two implementations serve different purposes: box_format resolves ambiguous
  input into internal-or-external IDs, queries only deals with internal IDs.

### C. Shared Metadata Field Dispatch Table

**Idea:** The pattern → action mapping (`@digest` → set markl, `!type` → set
type, identifier → add tag, `key=value` → add field) is similar in both
box_format and queries.

**Status:** Not explored in depth. The handlers produce fundamentally different
outputs (box_format populates `*sku.Transacted` metadata, queries builds
expression trees), so the sharing would be limited to the match patterns
themselves — which are already shared via `doddish.TokenMatcherType`,
`doddish.TokenMatcherBlobDigest`, etc.

### D. organize_text → box_format Delegation — ALREADY CLEAN

organize_text's `reader.readOneObj()` delegates all object format parsing to
`box_format.ReadStringFormat()`. No duplication exists here.

### E. inventory_list_coders/doddish — ALREADY CLEAN

`inventory_list_coders.doddish` is a thin wrapper that calls
`box_format.EncodeStringTo()` and `box_format.ReadStringFormat()`. No
duplication.

## Conclusions

The apparent redundancy between box_format and queries is mostly structural
similarity (both use doddish scan loops with pattern matching) rather than actual
code duplication. The key shared primitives — `doddish.Scanner`, `doddish.Seq`,
`TokenMatcher` patterns, and `ids.ObjectId.SetWithSeq()` — are already factored
into reusable components.

The two TODO comments about converting to decision trees may be better addressed
independently within each package as internal readability improvements rather
than as a shared abstraction.

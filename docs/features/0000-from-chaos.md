---
date: 2026-04-01
promotion-criteria: "the unit type ! is a content-addressed WASM blob whose
  functions are callable by the Go runtime via the primordial ABI; a meta-type
  (!toml-type-v2) satisfies type-implementable via its own WASM blob;
  a concrete type (!task) exports the 'actionable' interface with WATI function
  signatures; a conforming type (!vtodo) implements 'actionable' by referencing
  a codec WASM blob and TOML config; the projection pipeline resolves
  project_field through the full dispatch chain (Chaos → ! → meta-type → type →
  conforming type → codec); a CalDAV VTODO round-trips through
  checkin/checkout with full iCalendar preservation; BATS tests verify field
  projection, action dispatch, conformance validation, and codec WATI
  verification"
status: exploring
---

# Type Interface Contracts

## Problem Statement

### The Immediate Problem: Lossy Compilation

Haustoria (FDR-0007) translates between dodder's internal representation and
external systems. The current CalDAV implementation destructures VTODOs during
compilation: SUMMARY becomes a dodder description, DESCRIPTION becomes a raw
blob, CATEGORIES become tags, and the type is hardcoded to `!task`. The original
iCalendar structure is lost.

CalDAV properties with no dodder mapping (PRIORITY, RRULE, VALARM, DTSTART,
X-\* extensions, SEQUENCE) are discarded on compilation and cannot be recovered
on decompilation. External clients that set these properties see them vanish
after a dodder sync cycle.

This is the surface problem. Fixing it requires the blob to become the source of
truth rather than flat metadata. But making the blob authoritative requires the
type system to know how to extract fields from arbitrary blob formats. And making
the type system extensible enough to handle arbitrary formats --- iCalendar
today, browser bookmarks tomorrow, Kanban board JSON next week --- requires a
type system that can define its own extension points.

### The Structural Problem: Types Are Names, Not Contracts

`!task` is a label. There is no mechanism for the type system to say "a `!vtodo`
blob *is* a valid task" --- the only way to make a CalDAV VTODO behave as a
task is to shred it into dodder's flat metadata. A bookmark, a Kanban card, and
a GitHub issue could all be "tasks" in the sense that they have a summary and a
completion state, but the type system cannot express this without flattening
each into the same metadata layout.

Types cannot declare what they *mean* --- what fields they expose, what actions
they support, and which other types are structurally compatible. Types cannot
export named capabilities ("actionable", "describable", "schedulable") that
other types implement. There is no polymorphism.

### The Foundational Problem: The Type System Cannot Define Itself

Every type object has a type blob (currently `!toml-type-v1`). But `!toml-type-v1`
is a hardcoded format --- the Go binary knows how to parse it, and the type
system cannot change what a type blob means without changing Go code. The type
system describes objects but cannot describe itself.

The null type `!` (FDR-0010) was introduced as a staging ground for tool blobs.
But `!` should be more than that. If dodder's ambition is a self-hosting type
system --- where the objects stored in the repo define the behavior used to
process them (as described in the dynamic type registries design) --- then `!`
is the foundation. The Big Bang. The universal Turing machine from which the
rest of the type system derives.

## Design

### Cosmology

The design has four layers. Each layer escapes the one below it, meaning it
creates a new world where the constraints of the lower layer are no longer
directly visible to authors working at the higher layer.

**Layer 0: Chaos.** The Go binary. The primordial substrate. It is not a type,
not an object, not content-addressed. It has no identity in the object graph. It
knows exactly one thing: how to load a blob as a WASM (WebAssembly) module and
call functions on it with a specific calling convention. In Ruby terms, Chaos is
the C runtime. In lambda calculus, it is the reduction rule itself. In physics,
it is the rules that make computation possible but is not itself a computed
thing.

**Layer 1: The Unit Type (`!`).** The Big Bang. `!` is the first object Chaos
can execute. It is a WASM blob with a content-addressed digest and a signature,
like every other object. Its meta-type is itself --- the fixed point where the
chain terminates. `!` defines what it means to be a type by exporting
"type-implementable" --- the interface that all meta-types must satisfy. This is
the one interface that cannot be defined by the type system because it *is* the
type system. But because `!` is a regular object with a digest, it can be
overridden. The dodder.net seed repo ships a default `!`. A user can replace it.
A future `!` could implement a richer type system, compiled from a different
language --- as long as it satisfies the primordial ABI, Chaos can load it.

**Layer 2: Meta-types.** Types whose blobs are WASM modules that satisfy
"type-implementable". `!toml-type-v2` is the first meta-type --- its WASM knows
how to interpret TOML type blobs, resolve field maps, dispatch to format codecs,
and validate conformance. `!type-lua-v1` could be another --- its WASM contains
a Lua VM and interprets Lua scripts as type definitions. Meta-types escape Layer
1: type authors at this level write TOML or Lua, not WASM.

**Layer 3: Types and interfaces.** Types defined by meta-type blobs. `!task` is
a TOML type blob interpreted by `!toml-type-v2`. It exports "actionable" --- a
named interface with field declarations and WATI (WebAssembly Type Interface)
function signatures. `!vtodo` is another TOML type blob that implements
"actionable" by referencing a codec WASM blob and providing declarative config.
Types at this level escape Layer 2: consumers interact with named interfaces
("actionable"), not TOML schemas or Lua scripts.

Each layer creates an escape from the one below:

- `!` escapes Chaos --- Go no longer defines types, `!` does
- `!toml-type-v2` escapes WASM --- type authors write TOML, not WASM
- `!task` escapes TOML --- consumers interact with "actionable", not schemas
- `!vtodo` escapes the interface abstraction --- it references a codec and
  provides config

But every layer's artifacts are objects in the graph. The iCal codec, the Lua
VM, `!` itself --- they all have digests, they're all lockable via FDR-0001,
they're all overridable from the seed repo. The only thing outside the graph is
Chaos.

### The Primordial ABI (Exploratory)

**This section describes the interface between Chaos and `!`. It is the most
uncertain part of this design.** The primordial ABI is the universal Turing
machine of this system --- it must be capable of implementing itself. Getting
it wrong constrains everything above it. Getting it right means it never needs
to change, because `!` can extend the type system without modifying the ABI.

The primordial ABI is the set of WASM function signatures that Chaos (the Go
binary) knows how to call on `!`'s blob. These are the axioms. Everything else
derives from them.

A candidate set:

``` text
// Does this type export the named interface?
fn exports(type_blob: &[u8], interface_name: &str) -> bool

// Project a field value from content through a type's exported interface
fn project(content: &[u8], type_blob: &[u8], iface: &str, field: &str) -> Vec<u8>

// Mutate content by executing a named action on an interface
fn execute(content: &[u8], type_blob: &[u8], iface: &str, action: &str) -> Vec<u8>

// Validate content against a type's constraints
fn validate(content: &[u8], type_blob: &[u8]) -> Result<(), String>

// Extract references from content given its type
fn extract_refs(content: &[u8], type_blob: &[u8]) -> Vec<Reference>

// Verify that a WASM blob satisfies declared WATI signatures
fn verify_wati(wati_sigs: &[u8], wasm_blob: &[u8]) -> bool
```

Six functions. `exports`, `project`, `execute`, `validate`, `extract_refs`, and
`verify_wati`. Every type, every interface, every format codec derives from
these six entry points being callable on `!`'s blob.

`verify_wati` deserves special attention. It is how `!` verifies that a
downstream WASM blob actually satisfies the function signatures a type claims it
does. Without it, conformance is trust-based. With it, the type system can
validate itself at commit time. WASM module type signatures are statically
inspectable (the WASM binary format includes a type section), so signature
conformance can be checked without execution. Behavioral conformance ---
whether the function does the right thing --- cannot be checked statically, only
tested.

**Open concern: self-implementation.** The primordial ABI must be powerful enough
for `!` to define a type system that can redefine `!` itself. This means the six
functions must be sufficient to express the concept of "loading a WASM blob and
calling functions on it" --- because that's what `!` does to meta-types, and a
replacement `!` would need to do the same thing. The ABI must be a fixed point:
a WASM module called via these six functions must be able to implement a system
that calls other WASM modules via these same six functions. Whether the
candidate set above achieves this is not yet proven. It may need a seventh
function for WASM module instantiation, or `project`/`execute` may need to be
generalized into a single `call` with a richer dispatch protocol. This requires
prototyping.

**Open concern: host functions.** When `!`'s WASM needs to call into downstream
WASM blobs (meta-types calling codec blobs), does it go through Chaos (Go
instantiates the child WASM) or does it do so directly (WASM-to-WASM linking)?
The former is simpler and maintains Chaos as the only WASM host, but adds
call-stack overhead. The latter is more efficient but requires a WASM component
model that may not be mature enough. Phase 1 should use Chaos as intermediary.

**Open concern: the ABI is the one thing that isn't content-addressed.** It
lives in the Go binary. It's the Planck constant --- you can build a universe
on top of it, but you can't derive it from within that universe. If the ABI
ever needs to change, every `!` blob in every repo must be compatible with both
the old and new ABI, or a migration is required. This is the strongest argument
for getting it minimal and right.

### The Ruby Analogy

The design draws explicit inspiration from Ruby's metaclass architecture, with
the goal of being less dynamic and safer --- commit-time validated rather than
runtime `method_missing`.

  ---------------------------------------------------------------------------
  Ruby                          dodder
  ----------------------------- ---------------------------------------------
  C runtime                     Chaos (Go binary)

  `BasicObject` (defined in C)  `!` (WASM blob, implements primordial ABI)

  `Class` (enables `class`      `!toml-type-v2` (WASM, enables TOML type
  keyword)                      declarations)

  `Module` (mixin mechanism)    exported interfaces ("actionable",
                                "describable", "schedulable")

  `include Actionable`          `[implements.actionable]`

  `class Task; end`             `!task` (TOML type blob, exports "actionable")

  `Task.new`                    a `!vtodo` blob on a `!vtodo`-typed object

  `respond_to?(:complete)`      `Exports(!vtodo, "actionable")` → true

  `obj.send(:complete)`         `ExecuteAction(blob, !vtodo, "actionable",
                                "complete")`

  `method_missing` (runtime)    conformance validation (commit-time, never
                                runtime)
  ---------------------------------------------------------------------------

The key difference: Ruby's dynamism means `method_missing` is always possible.
Dodder validates conformance at commit time. If a type declares
`[implements.actionable]`, the finalizer verifies that every required field has a
mapping, every action has an implementation, and every referenced WASM blob
satisfies its declared WATI signatures. No runtime surprises.

### Interface Contracts

A type can declare one or more **interface contracts** --- named capabilities it
exports. Interfaces use capability names, not type names. `!task` exports
"actionable", not "task-like". The name describes what the interface enables,
not what the type is.

`!task`'s type blob declares the "actionable" interface:

``` toml
# !task type blob (interpreted by !toml-type-v2)
file-extension = "md"
vim-syntax-type = "markdown"

[exports.actionable]
description = "Things that can be started, completed, and tracked"

[[exports.actionable.fields]]
name = "summary"
kind = "string"
required = true

[[exports.actionable.fields]]
name = "status"
kind = "enum"
values = ["needs-action", "in-progress", "completed", "cancelled"]
default = "needs-action"

[[exports.actionable.fields]]
name = "due"
kind = "timestamp"
required = false

[[exports.actionable.fields]]
name = "priority"
kind = "int"
range = [0, 9]
required = false

[[exports.actionable.fields]]
name = "tags"
kind = "string-list"
required = false

# WATI function signatures — what a conforming codec must implement
[[exports.actionable.wati]]
name = "project_field"
params = ["blob:bytes", "field_name:string"]
returns = "field_value:bytes"

[[exports.actionable.wati]]
name = "execute_action"
params = ["blob:bytes", "action_name:string"]
returns = "mutated_blob:bytes"

# Declared actions — named mutations available on this interface
[exports.actionable.actions.complete]
description = "Mark the task as completed"
mutates = ["status"]

[exports.actionable.actions.reopen]
description = "Reopen a completed task"
mutates = ["status"]
```

The interface contract has three parts: **fields** (the shape of the data),
**WATI signatures** (the function contract that codec blobs must satisfy), and
**actions** (named mutations). Fields and actions are declarative data. WATI
signatures are the bridge to executable code.

A type can export multiple interfaces. A `!calendar-event` type might export
both "schedulable" (has start, end, recurrence) and "describable" (has summary,
body, tags). A `!task` might export "actionable" and "describable". Interfaces
are composable --- they are Ruby modules, not single inheritance.

### WATI: WebAssembly Type Interface

WATI is the mechanism by which declarative type definitions connect to
executable code. A type blob declares WATI function signatures as part of an
exported interface. A conforming type references a WASM blob (the **codec**)
that implements those signatures.

The WATI signatures in the `!task` example above say: any codec that wants to
make a blob "actionable" must export `project_field(blob, field_name) → value`
and `execute_action(blob, action_name) → mutated_blob`. The signatures are
format-agnostic --- the same signatures work for an iCal codec, a JSON codec, a
TOML codec, or a proprietary binary codec.

The codec WASM blob is a shared library. The iCal codec referenced by `!vtodo`
is the same blob referenced by `!vevent`, `!vjournal`, or any other
iCalendar-backed type. It's content-addressed, immutable, reusable. Its WASM
exports match the WATI signatures declared by whichever interface it's
satisfying. The TOML config section in the conforming type's blob is what
specializes each usage --- the codec is generic, the config makes it specific.

At commit time, `verify_wati` (one of `!`'s primordial ABI functions) checks
that the codec blob's WASM exports match the declared WATI signatures. This is
static verification --- WASM binaries include a type section that lists exported
function signatures. Behavioral correctness (does `project_field("SUMMARY")`
actually return the SUMMARY?) cannot be verified statically and is the domain of
tests.

### Conformance Declarations

A type that stores blob content in a format different from its interface's
native expectations declares **conformance** --- explicit inclusion, not
structural duck typing. `!vtodo` is a type whose blob content is raw iCalendar
and which declares conformance to the "actionable" interface exported by `!task`:

``` toml
# !vtodo type blob (interpreted by !toml-type-v2)
file-extension = "ics"
vim-syntax-type = "ical"
blob-format = "ical-vtodo"

[implements.actionable]
# Which type defines this interface
type-ref = "!task"

# WASM blob that satisfies the WATI sigs declared by !task's "actionable"
codec = "<@blake2b256-abc... !wasm"

# Declarative config passed TO the codec at call time
# The codec reads this to know how to map between iCal and interface fields
[implements.actionable.config.field-map]
summary = "SUMMARY"
status = "STATUS"
due = "DUE"
priority = "PRIORITY"
tags = "CATEGORIES"

[implements.actionable.config.status-map]
needs-action = "NEEDS-ACTION"
in-progress = "IN-PROCESS"
completed = "COMPLETED"
cancelled = "CANCELLED"

[implements.actionable.config.actions.complete]
set = { STATUS = "COMPLETED", PERCENT-COMPLETE = "100" }

[implements.actionable.config.actions.reopen]
set = { STATUS = "NEEDS-ACTION", PERCENT-COMPLETE = "0" }
```

The codec WASM blob receives the raw blob bytes *and* the TOML config as
arguments. The config is what the `!toml-type-v2` meta-type understands ---
it's the declarative escape from WASM. The iCal codec doesn't hardcode
"SUMMARY maps to summary". It reads the `field-map` from config and uses it.
A different type could map SUMMARY to a field called "title" by providing
different config to the same codec.

### Blob as Source of Truth

When a type declares conformance via `[implements]`, the blob becomes the
canonical source of truth for all mapped fields. Dodder metadata fields
(description, tags) become **cached projections** --- derived from the blob,
not independently authoritative.

This changes the data flow:

**Current (destructive compilation):**

    CalDAV VTODO → extract fields → store as flat metadata + raw blob
                                    (iCal structure lost)

**Proposed (preserving compilation):**

    CalDAV VTODO → store as !vtodo-typed blob → project fields on demand
                   (full iCal preserved)         (via interface contract)

The `CompileResult` in the haustoria interface reflects this:

``` go
type CompileResult struct {
    ExternalId string
    Blob       []byte
    BlobType   string   // "!vtodo" — the type system resolves the rest
    ETag       string
}
```

Metadata fields like description are still populated --- they're computed from
the blob via the interface contract's field map and cached on the object. This
keeps `dodder status`, `dodder show`, and query-by-description working without
every consumer needing to understand iCalendar. But the blob is the authority:
if the cached description and the blob's SUMMARY diverge, the blob wins.

### The Dispatch Chain

When the system needs a field defined by an interface contract, the full
resolution chain traverses the cosmology:

``` text
ProjectField(vtodo_blob, "actionable", "summary")
 │
 ├─ 1. Object type is !vtodo
 ├─ 2. !vtodo's meta-type is !toml-type-v2
 ├─ 3. !toml-type-v2's meta-type is !  (terminus)
 │
 │  Chaos calls !'s WASM (primordial ABI: project)
 │   │
 │   ├─ 4. ! sees that !toml-type-v2 satisfies "type-implementable"
 │   ├─ 5. ! delegates to !toml-type-v2's WASM
 │   │
 │   │  !toml-type-v2's WASM interprets the TOML type blob
 │   │   │
 │   │   ├─ 6. Parses !vtodo's type blob as TOML
 │   │   ├─ 7. Finds [implements.actionable]
 │   │   ├─ 8. Finds codec = <@blake2b256-abc... !wasm>
 │   │   ├─ 9. Finds config.field-map.summary = "SUMMARY"
 │   │   │
 │   │   │  !toml-type-v2 calls the codec WASM with config
 │   │   │   │
 │   │   │   ├─ 10. Codec receives (blob=ical_bytes, field="SUMMARY")
 │   │   │   ├─ 11. Codec parses iCal, extracts SUMMARY property
 │   │   │   └─ 12. Returns "Buy groceries"
 │   │   │
 │   │   └─ 13. !toml-type-v2 applies status-map if enum, returns value
 │   │
 │   └─ 14. ! returns the projected value to Chaos
 │
 └─ Result: FieldValue{Kind: String, Value: "Buy groceries"}
```

For action execution:

``` text
ExecuteAction(vtodo_blob, "actionable", "complete")
 │
 ├─ Same chain through ! → !toml-type-v2 → !vtodo type blob
 ├─ !toml-type-v2 reads config.actions.complete
 │    set = { STATUS = "COMPLETED", PERCENT-COMPLETE = "100" }
 ├─ Calls codec WASM: execute_action(blob, "complete")
 │    with set-instructions from config
 ├─ Codec parses iCal, sets STATUS=COMPLETED + PERCENT-COMPLETE=100
 ├─ Codec serializes modified iCal
 └─ Returns mutated blob bytes (full iCal preserved, only mapped props changed)
```

For a Lua-defined type, the chain differs at Layer 2:

``` text
ProjectField(smart_task_json, "actionable", "summary")
 │
 ├─ 1. Object type is !smart-task
 ├─ 2. !smart-task's meta-type is !type-lua-v1
 ├─ 3. !type-lua-v1's meta-type is !
 │
 │  ! delegates to !type-lua-v1's WASM (contains Lua VM)
 │   │
 │   ├─ 4. Loads !smart-task's type blob into Lua VM
 │   ├─ 5. Lua calls project_field("actionable", "summary", content)
 │   ├─ 6. Lua script parses JSON, applies custom logic
 │   └─ 7. Returns computed value
```

The same primordial ABI, the same `project` function on `!`, but the meta-type
determines how the type blob is interpreted. TOML type blobs go through the TOML
engine. Lua type blobs go through the Lua engine. The interface consumer doesn't
know or care.

### Interaction with Haustoria

The haustoria compilation model (FDR-0007) benefits directly. Today, each
haustoria implementation destructures external formats into dodder's flat
metadata. With interface contracts, the haustoria stores the external format as
a typed blob and lets the type system handle projection.

**CalDAV haustoria changes:**

-   `Compile` stores the raw VTODO iCalendar as a `!vtodo`-typed blob instead of
    extracting fields into `CompileResult.Description`, `.Tags`, etc.
-   `Decompile` reads the `!vtodo` blob directly and PUTs it to CalDAV, rather
    than reconstructing a VTODO from flat metadata fields.
-   `QueryCheckedOut` projects `summary` and `status` from the blob via the
    "actionable" interface for display in `dodder status`.
-   Actions like `complete` modify the blob through the interface rather than
    setting flat metadata fields.

**New haustoria implementations benefit immediately.** A future
`haustoria_chrest` (browser bookmarks) could define `!bookmark` implementing a
"linkable" interface. A `haustoria_nebulous` (NewsBlur articles) could define
`!article` implementing "describable". Each stores its native format and
declares conformance rather than writing bespoke field extraction code.

### Interaction with Typed Blob References

FDR-0001 Phase 3 requires every blob reference to carry a type lock. With
interface contracts, the type lock serves double duty:

-   It identifies the blob's format (how to parse it)
-   It identifies the blob's interface conformance (what capabilities are
    available)

A task object's blob reference becomes `<@blake2b256-... !vtodo@sig`. The type
lock `!vtodo@sig` pins both the format and the conformance declaration at commit
time. If `!vtodo`'s field mappings change in a later version, existing objects
retain the old mapping through the signature lock.

### Interaction with Three-Way Merge

FDR-0007's three-way merge operates on hyphence text. With blob-as-truth, the
merge target changes:

-   **Metadata-level merges** still use hyphence (tags, referenced objects, blob
    reference list). These are structural and format-independent.
-   **Blob-level merges** operate on the blob's native format. For iCalendar,
    this means merging iCal property lines rather than hyphence-rendered
    summaries.

For Phase 1, the blob is treated as opaque at the merge level ---
last-writer-wins for the blob, with metadata merges remaining three-way.
Semantic blob merging (per-property iCal merge via a format-aware diff codec) is
a natural extension: the "mergeable" interface, with WATI signatures for
`diff(base, ours, theirs) → merged`. But this is future work.

### Conformance Validation

At commit time, the finalizer validates conformance declarations:

-   Every required interface field must have a mapping in `field-map` (or be
    handled by the codec)
-   Every action declared by the interface must have either a `set` block or a
    codec that exports the corresponding WATI function
-   `verify_wati` confirms the referenced codec WASM blob actually exports the
    functions declared in the interface's WATI signatures
-   Status maps (for enum fields) must cover all values in the interface's
    `values` list

Validation failures are hard errors at commit, matching FDR-0001's hard-fail
policy for missing type locks on blob references.

At runtime, projection failures (blob doesn't contain the mapped property, value
not coercible to declared kind) are soft errors --- the field returns a zero
value and the error is logged, but the operation continues. This accommodates
real-world iCalendar where optional properties like DUE and PRIORITY may be
absent.

## Design Decisions

1.  **Chaos is not an object.** The Go binary and the primordial ABI are the one
    thing outside the content-addressed graph. This is deliberate: you need
    something outside the system to bootstrap the system. Chaos is the Planck
    constant --- you can build a universe on it, but you can't derive it from
    within that universe. The tradeoff: if the primordial ABI ever needs to
    change, a migration is required across all repos.

2.  **`!` is an object, not a sentinel.** Unlike FDR-0010's original framing of
    `!` as "no blob, no definition, no validation", this design makes `!` a
    content-addressed WASM blob with a digest and signature. It can be locked,
    referenced, overridden. Rationale: if `!` defines the type system, it must
    be versionable and replaceable, or the type system ossifies. The seed repo
    provides the default `!`; users or organizations can substitute their own.
    This FDR supersedes FDR-0010's definition of `!` from "no blob, no
    definition" to "content-addressed WASM blob that defines the type system".

3.  **Interfaces are named capabilities, not type names.** `!task` exports
    "actionable", not "task-like". The name describes what the interface enables
    ("things that can be started, completed, and tracked"), not what the
    exporting type is. Rationale: the same capability can be exported by
    unrelated types. "describable" might be exported by `!task`, `!note`,
    `!bookmark`, and `!article`. Naming interfaces after capabilities encourages
    composition; naming them after types encourages inheritance hierarchies.

4.  **Explicit conformance over structural typing.** A type must declare
    `[implements.actionable]` to conform. Having the right fields is not
    sufficient. Rationale: explicit declaration means the type author has
    verified the mapping, the commit-time validator can check it, and the
    dispatch chain knows exactly where to look. Structural typing would make
    conformance fragile and implicit --- Ruby's `respond_to?` without the safety
    of `include`.

5.  **WATI signatures in the interface, codec blobs in the conformance.**
    The interface declares *what* functions are needed (signatures). The
    conforming type declares *which* WASM blob provides them (codec reference).
    Rationale: this separates the contract from the implementation. Multiple
    conforming types can share the same codec blob (the iCal codec serves
    `!vtodo`, `!vevent`, `!vjournal`) while each provides different config.

6.  **Declarative config as the TOML escape from WASM.** The `[implements.X.config]`
    section is TOML that the meta-type's WASM passes to the codec at call time.
    Type authors write TOML field-maps and status-maps; they don't write WASM.
    Rationale: this is the escape chain in action. `!toml-type-v2`'s WASM
    interprets TOML so that downstream authors don't need WASM. The declarative
    config is the userland; WASM is the kernel.

7.  **Metadata as cached projection, not independent truth.** When a type
    declares `[implements]`, mapped metadata fields are derived from the blob.
    Rationale: two sources of truth diverge. The blob is the richer
    representation. The cache is populated eagerly at compile time (Phase 1);
    lazy projection is a future optimization keyed on blob digest + type
    signature.

8.  **Last-writer-wins for blob merges in Phase 1.** Format-aware blob diffing
    (per-property iCal merge) is deferred. Rationale: significant engineering
    effort, not required for initial CalDAV round-trip where concurrent blob
    edits are rare. Metadata merges (tags, references) remain three-way.

9.  **`verify_wati` as static signature check.** WASM binaries include a type
    section listing exported function signatures. Conformance checking inspects
    this section at commit time without executing the WASM. Behavioral
    correctness is not verified statically --- it's the domain of BATS tests
    and integration testing.

## Implementation Order

### Phase 1: Foundation

1.  Redefine `!` as a content-addressed blob (WASM module) with digest and
    signature. Update `bravo/ids/type.go`, box/hyphence/binary codecs to handle
    `!` as a real object reference rather than a hardcoded sentinel.
2.  Primordial ABI prototype: Go-side WASM host (wazero or wasmtime) that can
    load `!`'s blob and call the six candidate functions. Verify the ABI is
    sufficient for `!` to delegate to a child WASM (meta-type).
3.  `!toml-type-v2` WASM module: interprets TOML type blobs, dispatches to codec
    blobs. Satisfies "type-implementable".

### Phase 2: Interface System

4.  `[exports]` section in type blob TOML schema: field declarations, WATI
    signatures, action declarations.
5.  `[implements]` section: codec reference, config (field-maps, status-maps,
    action definitions).
6.  iCalendar codec WASM blob: parses RFC 5545, implements `project_field` and
    `execute_action` WATI signatures, driven by TOML config.
7.  Projection pipeline: dispatch chain from Chaos → `!` → meta-type → codec.
8.  `verify_wati` implementation: static WASM type section inspection at commit
    time.
9.  Conformance validation in `object_finalizer`.

### Phase 3: CalDAV Integration

10. `!task` type object with `[exports.actionable]`.
11. `!vtodo` type object with `[implements.actionable]`, referencing the iCal
    codec.
12. Modify `haustoria_caldav.Compile` to store raw iCal as `!vtodo`-typed blob.
13. Modify `haustoria_caldav.Decompile` to read `!vtodo` blob directly.
14. Modify `QueryCheckedOut` to project fields via "actionable" interface.
15. Action dispatch: `complete` and `reopen`.
16. BATS integration tests: full-fidelity CalDAV VTODO round-trip, field
    projection, action execution, property preservation.

### Phase 4: Extension

17. `!type-lua-v1` meta-type (WASM containing Lua VM).
18. Additional interfaces: "describable", "schedulable", "linkable".
19. Additional codecs: JSON, TOML, markdown-frontmatter.
20. Seed repo packaging for `!`, meta-types, and builtin codecs.

## Open Questions

-   **Is the candidate primordial ABI sufficient for self-implementation?** The
    six functions must let `!` define a type system that can redefine `!`. This
    is the most critical open question. It may need a seventh function for WASM
    module instantiation, or `project`/`execute` may need to be generalized into
    a single `call` with a richer dispatch protocol. Prototyping is required.

-   **Host functions vs. self-contained WASM.** When a meta-type's WASM needs to
    call a codec WASM, does it go through Chaos (Go instantiates the child WASM)
    or directly (WASM-to-WASM linking, component model)? Phase 1 should use
    Chaos as intermediary, but the long-term answer depends on the WASM component
    model's maturity.

-   **Multiple interface conformance and name collisions.** A type can implement
    multiple interfaces (`[implements.actionable]`, `[implements.schedulable]`).
    If two interfaces define a field with the same name but different kinds, which
    wins? Options: (a) error at commit time, (b) fully qualified field names
    (`actionable.summary` vs. `schedulable.summary`), (c) the type author
    resolves the conflict in config.

-   **Cache invalidation for projections.** If a `!vtodo` type blob is updated
    with a new field mapping, cached projections on existing objects are stale.
    The signature lock (FDR-0001) pins the type version per object, so the
    stale projection is still *correct for that version*. But queries against
    the latest type definition may return inconsistent results until objects are
    re-committed. Is this acceptable, or does projection need to be
    version-aware?

-   **Granularity of codec WASM blobs.** One blob per format (iCal codec handles
    all iCal types)? One per format-interface pair (iCal-actionable codec, iCal-
    schedulable codec)? One universal codec that reads its behavior from config?
    The first is simplest; the last is most flexible.

-   **Action composition with two-stage commit.** An action mutates a blob,
    changing its digest, requiring a new commit. FDR-0006's plan phase would
    need action execution as a step. Does the builder need an action-aware
    operation type alongside create/update/delete?

-   **How does `!` bootstrap?** If `!` is an object in the store, it needs to
    exist before any other object can be committed (since commit requires type
    validation). But creating `!` is itself a commit. The genesis sequence must
    special-case `!`'s creation --- Chaos writes it directly without type
    validation, then all subsequent commits go through the normal path. This
    is the cosmological singularity: the one moment where Chaos acts without
    `!`'s mediation.

-   **Can the primordial ABI be discovered rather than hardcoded?** If Chaos
    could read the ABI from a well-known location in the WASM module's custom
    sections (WASM supports named custom sections), the ABI becomes partially
    self-describing. `!`'s WASM would declare "these are my entry points" in a
    custom section, and Chaos would read that section to know what to call.
    This pushes more of the ABI definition into content-addressed space, but
    there's a turtles-all-the-way-down problem: something must define the
    format of the custom section, and that something is Chaos.

## Related

-   [FDR-0001: Object Locks](0001-object-locks.md) --- typed blob references
    with signature locks; codec WASM blobs and conformance declarations are
    pinned via type locks at commit time
-   [FDR-0007: Pluggable Checkout Stores](0007-checkout-bridges.md) --- haustoria
    architecture, compilation model, three-way merge; interface contracts
    eliminate the destructuring step in compilation
-   [FDR-0010: Core Types](0010-core-types.md) --- original null type design;
    this FDR supersedes `!`'s definition from "no blob, no definition" to
    "content-addressed WASM blob that defines the type system"
-   [Dynamic Type Registries](../plans/2026-02-23-dynamic-type-registries.md) ---
    vision for types-as-objects with runtime behavior; this FDR provides the
    concrete mechanism (WATI, codec blobs, meta-types)
-   [Typed Blob References](../plans/2026-03-21-typed-blob-references-design.md) ---
    blob type locks that carry format and conformance identity; WATI verification
    uses these locks to locate codec blobs

# Research: Abstracting the organize_text Package

This document analyzes the `organize_text` package and identifies the changes needed to abstract it from `sku.SkuType` to support other element types.

## Executive Summary

The `organize_text` package (`src/papa/organize_text/`) is tightly coupled to the SKU system. To abstract it for generic use, approximately **15-20 interface definitions** and **significant refactoring** across **17 files** would be required. The core challenge is that the package deeply interweaves hierarchical organization logic with SKU-specific metadata operations.

---

## 1. Current Architecture

### 1.1 Package Location and Purpose

**Path:** `src/papa/organize_text/`

**Purpose:** Creates a textual, markdown-like representation of objects organized hierarchically by tags. Users can edit this in a git-rebase-like manner to reorganize, add tags, change descriptions, etc.

### 1.2 File Structure

| File | Lines | Primary Responsibility |
|------|-------|----------------------|
| `main.go` | 137 | Core `Text` struct, `New()`, `Refine()`, `ReadFrom()`, `WriteTo()` |
| `structs.go` | 139 | Internal `obj` wrapper, `Objects` collection |
| `options.go` | 251 | `Options` and `Flags` configuration |
| `assignment.go` | 365 | Hierarchical tree node structure |
| `constructor.go` | 316 | Tree construction from SKU sets |
| `constructor2.go` | ~100 | Alternative constructor (experimental) |
| `metadata.go` | 164 | Header metadata handling |
| `reader.go` | 325 | Parsing organize text format |
| `writer.go` | 113 | Serializing to organize text format |
| `refiner.go` | 305 | Tree optimization (prefix joints, merging) |
| `changeable.go` | 177 | Converting tree back to SKU set with changes |
| `changes.go` | 229 | Before/After change tracking |
| `set_prefix_transacted.go` | 259 | Tag prefix grouping |
| `option.go` | 240 | OptionComment system |
| `errors.go` | ~30 | Error types |

---

## 2. SKU Coupling Analysis

### 2.1 Direct Type Dependencies

The package imports and uses these SKU-related types:

```go
// From kilo/sku package
sku.SkuType           // = *CheckedOut (type alias)
sku.SkuTypeSet        // = CheckedOutSet
sku.SkuTypeSetMutable // = CheckedOutMutableSet
sku.ObjectFactory     // Factory for creating/cloning SKU objects
sku.Transacted        // Core versioned object type
sku.TransactedResetter // Pool-safe reset operations
sku.Query             // Query type for matchers
sku.TransactedMutableSet
sku.ExternalObjectId
```

### 2.2 Coupling Points by File

#### `structs.go` - Core Object Wrapper

```go
type obj struct {
    sku  sku.SkuType        // SKU type reference
    tipe tag_paths.Type     // Direct vs Implicit tag type
}

// Methods delegating to SKU
func (o obj) GetObjectId() *ids.ObjectId { return o.sku.GetObjectId() }
func (o obj) GetSku() *sku.Transacted { return o.sku.GetSku() }
func (o obj) GetSkuExternal() *sku.Transacted { return o.sku.GetSkuExternal() }
func (a *obj) GetExternalObjectId() sku.ExternalObjectId { ... }
func (a *obj) cloneWithType(t tag_paths.Type) *obj { ... sku.CloneSkuType(a.sku) ... }
```

#### `assignment.go` - Tree Node

```go
type Assignment struct {
    sku.Transacted           // EMBEDDED - used for tag storage on nodes
    IsRoot  bool
    Depth   int
    objects map[string]struct{}
    Objects
    Children []*Assignment
    Parent   *Assignment
}
```

The embedding of `sku.Transacted` is particularly significant - it uses the SKU's metadata structure to store tags on tree nodes, even though nodes don't represent actual SKUs.

#### `options.go` - Configuration

```go
type Options struct {
    Skus sku.SkuTypeSet      // Input objects
    sku.ObjectFactory        // Factory for creating SKU instances
    // ...
}
```

#### `changeable.go` - Output Reconstruction

```go
var keyer = sku.GetExternalLikeKeyer[sku.SkuType]()

func (ot *Text) GetSkus(original sku.SkuTypeSet) (out SkuMapWithOrder, err error)
func (assignment *Assignment) addToSet(ot *Text, output SkuMapWithOrder, objectsFromBefore sku.SkuTypeSet) error
```

This file contains the logic for reconstructing the final SKU set from the edited tree - it applies tag changes, type changes, description changes back to the objects.

#### `set_prefix_transacted.go` - Grouping Logic

```go
func (prefixSet *PrefixSet) AddSku(object sku.SkuType) error
// Accesses: object.GetState(), object.GetSkuExternal().GetMetadata().GetIndex()...
```

#### `reader.go` - Parsing

```go
type reader struct {
    options Options  // Contains ObjectFactory
    // ...
}

func (r *reader) readOneObj(rb *catgut.RingBuffer, t tag_paths.Type) error {
    z.sku = assignmentReader.options.ObjectFactory.Get()
    // Uses fmtBox.ReadStringFormat() which is SKU-aware
    sku.TransactedResetter.ResetWith(z.GetSku(), z.GetSkuExternal())
}
```

#### `writer.go` - Serialization

```go
type writer struct {
    sku.ObjectFactory
    objects.Metadata
    // ...
}

func (av writer) write(a *Assignment) error {
    // Uses fmtBox.EncodeStringTo() which is SKU-aware
    cursor := object.sku.Clone()
    cursorExternal := cursor.GetSkuExternal()
    cursorExternal.GetMetadataMutable().Subtract(av.Metadata)
}
```

### 2.3 External SKU Dependencies

The package also depends on:

- `box_format.BoxCheckedOut` - SKU-aware serialization format
- `ids.TagSet`, `ids.TagStruct` - Tag system (potentially reusable)
- `ids.ObjectId`, `ids.ZettelId` - SKU identification
- `ids.TypeStruct` - Type system
- `ids.RepoId` - Repository identification
- `ids.Abbr` - Abbreviation system
- `checked_out_state.State` - Object lifecycle states
- `objects.Metadata`, `objects.MetadataMutable` - Metadata structures
- `tag_paths.Type` - Direct vs Implicit tag classification

---

## 3. Required Abstractions

### 3.1 Core Element Interface

To abstract away from `sku.SkuType`, a generic element interface would need to be defined:

```go
// Proposed interface for abstractable elements
type OrganizableElement interface {
    // Identification
    GetKey() string           // Unique key for deduplication
    GetDisplayId() string     // ID shown in organize text

    // Metadata access
    GetTags() ids.TagSet
    GetDescription() string
    GetType() string

    // Mutability
    Clone() OrganizableElement
    SetTags(ids.TagSet) error
    SetDescription(string) error
    SetType(string) error
    AddTag(ids.TagStruct) error

    // State (for filtering)
    GetState() ElementState
    SetState(ElementState) error

    // Serialization hooks
    fmt.Stringer
}
```

### 3.2 Element Factory Interface

```go
type ElementFactory[T OrganizableElement] interface {
    Get() T
    ResetWith(dst, src T)
    Clone(T) T
    SetDefaultsIfNecessary()
}
```

### 3.3 Element Set Interface

```go
type ElementSet[T OrganizableElement] interface {
    Add(T) error
    Get(key string) (T, bool)
    Del(key string) error
    All() iter.Seq[T]
    Len() int
}
```

### 3.4 Element Formatter Interface

```go
type ElementFormatter[T OrganizableElement] interface {
    // Read element from text format
    ReadStringFormat(element T, reader io.RuneScanner) (int64, error)
    // Write element to text format
    EncodeStringTo(element T, writer io.StringWriter) (int64, error)
}
```

### 3.5 Tag Index Interface

The package heavily uses tag path indexing for grouping. This would need abstraction:

```go
type TagIndex interface {
    GetImplicitTags() ids.TagSet
    GetTagPaths() TagPathSet
}

type MetadataWithTagIndex interface {
    GetIndex() TagIndex
}
```

### 3.6 Node Metadata Holder

Currently `Assignment` embeds `sku.Transacted` for tag storage. This needs decoupling:

```go
type NodeMetadata interface {
    GetTags() ids.TagSet
    GetTagsMutable() ids.TagSetMutable
    SetTags(ids.TagSet)
}

// Could be implemented as:
type SimpleNodeMetadata struct {
    tags ids.TagSet
}
```

---

## 4. Files Requiring Modification

### 4.1 Heavy Modifications (8 files)

| File | Changes Required |
|------|-----------------|
| `structs.go` | Replace `sku.SkuType` with generic `T OrganizableElement` |
| `options.go` | Parameterize with generic element type, replace `sku.ObjectFactory` |
| `assignment.go` | Remove `sku.Transacted` embedding, use `NodeMetadata` interface |
| `constructor.go` | Parameterize, use generic element factory and set |
| `changeable.go` | Parameterize output types, generalize keying |
| `reader.go` | Use generic element formatter |
| `writer.go` | Use generic element formatter |
| `set_prefix_transacted.go` | Parameterize element type, use interfaces for tag access |

### 4.2 Moderate Modifications (4 files)

| File | Changes Required |
|------|-----------------|
| `main.go` | Propagate generic type parameter |
| `changes.go` | Parameterize `SkuMapWithOrder` to generic `ElementMapWithOrder` |
| `metadata.go` | Abstract `sku.Query` references |
| `refiner.go` | Use `NodeMetadata` interface instead of `sku.Transacted` methods |

### 4.3 Minor or No Modifications (5 files)

| File | Changes Required |
|------|-----------------|
| `constructor2.go` | Mirror changes from `constructor.go` |
| `option.go` | No changes needed (independent of SKU types) |
| `errors.go` | No changes needed |
| `reader_test.go` | Update to use test element type |

---

## 5. Proposed Abstraction Strategy

### 5.1 Option A: Full Generic Parameterization

Transform the package to use Go generics throughout:

```go
type Text[T OrganizableElement] struct {
    Options[T]
    *Assignment[T]
}

type Options[T OrganizableElement] struct {
    Elements ElementSet[T]
    Factory  ElementFactory[T]
    Formatter ElementFormatter[T]
    // ...
}
```

**Pros:**
- Type-safe at compile time
- No runtime type assertions
- Clean API

**Cons:**
- Massive refactoring effort (all 17 files)
- Breaking change for all consumers
- Go generics verbosity

### 5.2 Option B: Interface-Based Abstraction

Define interfaces and use them without full generic parameterization:

```go
type Text struct {
    Options
    *Assignment
}

type Options struct {
    Elements ElementSet      // interface type
    Factory  ElementFactory  // interface type
    Formatter ElementFormatter // interface type
}
```

**Pros:**
- Less invasive refactoring
- Easier migration path
- More idiomatic for Go 1.x codebases

**Cons:**
- Runtime type assertions in some places
- Loss of compile-time type safety at boundaries

### 5.3 Option C: Extract Core Logic (Recommended)

Create a new `organize_text_core` package with abstract interfaces, then make `organize_text` a thin wrapper that provides SKU-specific implementations:

```
src/lima/organize_text_core/   # New abstract package
    interfaces.go              # OrganizableElement, ElementFactory, etc.
    assignment.go              # Generic assignment tree
    refiner.go                 # Tag-based refinement
    reader_base.go             # Format parsing without element specifics
    writer_base.go             # Format writing without element specifics

src/papa/organize_text/        # Existing package, now a wrapper
    main.go                    # Instantiates organize_text_core with SKU types
    sku_element.go             # Implements OrganizableElement for SkuType
    sku_formatter.go           # Implements ElementFormatter for BoxCheckedOut
```

**Pros:**
- Backward compatible (existing API preserved)
- Clean separation of concerns
- Allows incremental adoption of generic version
- Other element types can directly use `organize_text_core`

**Cons:**
- Two packages to maintain
- Some code duplication in adapter layer

---

## 6. Specific Code Changes Required

### 6.1 Assignment Decoupling

**Current:**
```go
type Assignment struct {
    sku.Transacted  // Uses for tag storage
    // ...
}

func newAssignment(depth int) *Assignment {
    assignment := &Assignment{...}
    sku.TransactedResetter.Reset(&assignment.Transacted)
    return assignment
}
```

**Proposed:**
```go
type Assignment struct {
    tags     ids.TagSetMutable
    metadata NodeMetadata
    // ...
}

func newAssignment(depth int) *Assignment {
    return &Assignment{
        tags: ids.MakeTagSetMutable(),
        // ...
    }
}
```

### 6.2 Object Wrapper Generalization

**Current:**
```go
type obj struct {
    sku  sku.SkuType
    tipe tag_paths.Type
}
```

**Proposed:**
```go
type obj[T OrganizableElement] struct {
    element T
    tipe    tag_paths.Type
}
```

### 6.3 Options Generalization

**Current:**
```go
type Options struct {
    Skus sku.SkuTypeSet
    sku.ObjectFactory
    fmtBox *box_format.BoxCheckedOut
    // ...
}
```

**Proposed:**
```go
type Options[T OrganizableElement] struct {
    Elements ElementSet[T]
    Factory  ElementFactory[T]
    Formatter ElementFormatter[T]
    // ...
}
```

### 6.4 Reader/Writer Generalization

The `reader` and `writer` structs use `box_format.BoxCheckedOut` for serialization. This needs to be abstracted to a generic formatter interface that can handle different element types with their own serialization formats.

---

## 7. Dependencies to Preserve or Abstract

### 7.1 Can Be Preserved As-Is

- `ids.TagSet`, `ids.TagStruct`, `ids.TagSlice` - Tag system is generic
- `ids.TypeStruct` - Type system is generic
- `tag_paths.Type` - Direct/Implicit classification is generic
- `expansion.*` - Tag expansion is generic
- `format.LineWriter` - Output formatting is generic
- `catgut.*` - String scanning is generic
- `hyphence.*` - Metadata format is generic

### 7.2 Must Be Abstracted

- `sku.SkuType` -> `OrganizableElement` interface
- `sku.SkuTypeSet` -> `ElementSet` interface
- `sku.ObjectFactory` -> `ElementFactory` interface
- `sku.Transacted` (embedded in Assignment) -> `NodeMetadata` interface
- `box_format.BoxCheckedOut` -> `ElementFormatter` interface
- `sku.GetExternalLikeKeyer` -> Key function in `ElementFactory`
- `checked_out_state.State` -> Generic `ElementState`

### 7.3 Needs Careful Consideration

- `ids.ObjectId` - Used for sorting and display; may need abstraction or configuration
- `ids.RepoId` - Repository context; may not apply to all element types
- `ids.Abbr` - Abbreviation system; element-type specific
- `sku.Query` - Query system in metadata; may need abstraction

---

## 8. Estimated Effort

### 8.1 Interface Definition Phase
- Define core interfaces: ~200-300 lines
- Define helper types: ~100-150 lines
- **Estimate:** 1-2 days

### 8.2 Core Package Extraction (Option C)
- Extract `organize_text_core`: ~1500-2000 lines
- Create SKU adapter in `organize_text`: ~300-500 lines
- **Estimate:** 3-5 days

### 8.3 Testing and Migration
- Update existing tests: ~1 day
- Add generic tests: ~1-2 days
- Integration testing: ~1-2 days
- **Estimate:** 3-5 days

### 8.4 Total Estimated Effort
**7-12 days** for a complete abstraction following Option C

---

## 9. Alternative: Minimal Viable Abstraction

If full abstraction is too costly, consider a minimal approach:

1. **Extract just the tree structure** - Make `Assignment` not embed `sku.Transacted`
2. **Keep SKU serialization** - Continue using `box_format` but behind an interface
3. **Parameterize input/output** - Only abstract `ElementSet` and `ElementFactory`

This would reduce effort to approximately **3-5 days** while still enabling different input sources.

---

## 10. Recommendations

### For New Element Types

1. **Implement `OrganizableElement`** interface for your element type
2. **Implement `ElementFactory`** for creation/cloning
3. **Implement `ElementFormatter`** for serialization
4. **Use `organize_text_core`** (after extraction) directly

### For Existing Code

1. **Keep `organize_text` package** as the SKU-specific implementation
2. **Gradually migrate** internal code to use interfaces
3. **Maintain backward compatibility** through the adapter pattern

### Priority Order

1. **High:** Extract `Assignment` from `sku.Transacted` embedding
2. **High:** Define `OrganizableElement` interface
3. **Medium:** Parameterize `obj` wrapper struct
4. **Medium:** Abstract `ElementFormatter`
5. **Low:** Full generic parameterization of all types

---

## Appendix A: Files by Dependency Depth

```
Level 0 (No SKU deps):
  - errors.go
  - option.go (partial)

Level 1 (Light SKU deps):
  - refiner.go (via Assignment.Transacted)

Level 2 (Moderate SKU deps):
  - metadata.go (sku.Query in Matchers)
  - changes.go (SkuMapWithOrder)

Level 3 (Heavy SKU deps):
  - main.go (Options, Assignment)
  - structs.go (obj wraps SkuType)
  - options.go (SkuTypeSet, ObjectFactory)
  - assignment.go (embeds Transacted)
  - constructor.go (uses all SKU types)
  - changeable.go (keyer, factory, sets)
  - reader.go (ObjectFactory, fmtBox)
  - writer.go (ObjectFactory, fmtBox)
  - set_prefix_transacted.go (SkuType throughout)
```

## Appendix B: Method Calls on sku.SkuType

Methods called on `sku.SkuType` (i.e., `*CheckedOut`) throughout the package:

| Method | Usage Count | Purpose |
|--------|-------------|---------|
| `GetObjectId()` | 5 | Get object identifier |
| `GetSku()` | 3 | Get internal transacted |
| `GetSkuExternal()` | 25+ | Get external transacted (primary access) |
| `GetExternalObjectId()` | 2 | Get external object ID |
| `GetState()` | 3 | Check object state |
| `SetState()` | 3 | Update object state |
| `Clone()` | 5 | Clone for modification |
| `String()` | 3 | String representation |

On `*sku.Transacted` (via `GetSkuExternal()`):

| Method | Usage Count | Purpose |
|--------|-------------|---------|
| `GetMetadata()` | 15+ | Read metadata |
| `GetMetadataMutable()` | 10+ | Modify metadata |
| `ObjectId` (field) | 8 | Access object ID directly |
| `AddTag()` | 5 | Add tag to metadata |
| `RepoId` (field) | 3 | Access repo ID |
| `GetTai()` | 1 | Access timestamp |

This analysis shows that abstracting `GetSkuExternal()` and its return type is the critical path for successful abstraction.

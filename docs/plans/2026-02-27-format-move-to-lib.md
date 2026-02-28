# Move format Package to lib/delta Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move `internal/foxtrot/format` to `lib/delta/format`, eliminating its
sole `internal/` dependency on `string_format_writer.LenStringMax`.

**Architecture:** Replace the cross-tier constant reference with a local
constant, `git mv` the package from `internal/foxtrot/` to `lib/delta/`, then
update all 6 import sites. The package's dependencies (`lib/_/interfaces`,
`lib/bravo/errors`, `lib/charlie/ohio`) are all within delta's tier constraints.

**Tech Stack:** Go, NATO tier hierarchy (`lib/delta` depends on `lib/charlie`
and below)

---

## Task 1: Replace `string_format_writer.LenStringMax` with local constant

**Files:**
- Modify: `go/internal/foxtrot/format/strings.go`

**Step 1: Replace the import and constant reference**

Replace the entire file contents with:

```go
package format

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

const lenRightAlignMax = 17

func MakeFormatStringRightAligned(
	f string,
	args ...any,
) interfaces.FuncWriter {
	return func(w io.Writer) (n int64, err error) {
		f = fmt.Sprintf(f+" ", args...)

		diff := lenRightAlignMax + 1 - utf8.RuneCountInString(
			f,
		)

		if diff > 0 {
			f = strings.Repeat(" ", diff) + f
		}

		var n1 int

		if n1, err = io.WriteString(w, f); err != nil {
			n = int64(n1)
			err = errors.Wrap(err)
			return n, err
		}

		n = int64(n1)

		return n, err
	}
}
```

The constant `string_format_writer.LenStringMax` has value `17` (length of
`StringIndent`). We replace it with a private `lenRightAlignMax` constant and
remove the `internal/echo/string_format_writer` import.

**Step 2: Build to verify**

Run: `just build` from `go/`
Expected: Succeeds

**Step 3: Commit**

```
refactor(format): replace string_format_writer.LenStringMax with local constant
```

---

## Task 2: Move package from internal/foxtrot to lib/delta

**Files:**
- Move: `go/internal/foxtrot/format/` → `go/lib/delta/format/`
- Modify: 6 import sites

**Step 1: git mv the package**

```
git mv go/internal/foxtrot/format go/lib/delta/format
```

**Step 2: Update all 6 import sites**

Replace `code.linenisgreat.com/dodder/go/internal/foxtrot/format` with
`code.linenisgreat.com/dodder/go/lib/delta/format` in:

1. `go/internal/golf/triple_hyphen_io/coder_metadata.go`
2. `go/internal/juliett/object_metadata_fmt_triple_hyphen/formatter_components.go`
3. `go/internal/papa/organize_text/main.go`
4. `go/internal/papa/organize_text/metadata.go`
5. `go/internal/papa/organize_text/writer.go`
6. `go/internal/whiskey/local_working_copy/format_type.go`

**Step 3: Build to verify**

Run: `just build` from `go/`
Expected: Succeeds

**Step 4: Run unit tests**

Run: `just test-go` from `go/`
Expected: All pass

**Step 5: Commit**

```
refactor(format): move from internal/foxtrot to lib/delta
```

---

## Task 3: Update documentation

**Files:**
- Modify: `go/lib/delta/format/CLAUDE.md` (moved from internal)

**Step 1: Update CLAUDE.md**

The existing CLAUDE.md at `go/internal/foxtrot/format/CLAUDE.md` will have been
moved by `git mv`. Update it to reflect the full API:

```markdown
# format

Format writer utilities for type-safe I/O with composable writer combinators,
line-based reading/writing, and sequential write helpers.

## Key Functions

### Writer Combinators

- `MakeWriter[E]`: Create writer from format function and element
- `MakeWriterOr[A,B]`: Write first non-empty element (Stringer-based)
- `MakeWriterPtr[E]`: Writer for pointer types
- `MakeFormatString`: Printf-style formatted writer
- `MakeStringer`: Writer from any `fmt.Stringer`
- `MakeFormatStringer[E]`: Convert string formatter to writer
- `MakeFormatStringRightAligned`: Right-aligned formatted string
- `Write`: Sequentially execute multiple FuncWriters

## Line I/O

- `LineWriter`: Buffered line output with `WriteTo(io.Writer)`
  - `WriteLines()`, `WriteFormat()`, `WriteKeySpaceValue()`
  - `WriteEmpty()`, `WriteExactlyOneEmpty()`
- `MakeLineReaderConsumeEmpty`: Line reader skipping blanks
- `MakeLineReaderPassThruEmpty`: Line reader passing blanks
- `MakeDelimReaderConsumeEmpty`: Custom delimiter reader
- `ReadLines()`, `ReadSep()`: Convenience read functions
```

**Step 2: Commit**

```
docs: update format CLAUDE.md after move to lib/delta
```

---

## Task 4: Run full test suite

**Step 1: Build debug binaries**

Run: `just build` from `go/`

**Step 2: Run full test suite**

Run: `just test` from `go/`
Expected: All pass (unit + integration)

**Step 3: If tests fail, investigate and fix**

This is a pure move — all external call sites see identical signatures via
updated import paths. Failures would indicate a missed import update.

---

## Summary of Changes

| What | Before | After |
|------|--------|-------|
| Package location | `internal/foxtrot/format/` | `lib/delta/format/` |
| Import path | `.../internal/foxtrot/format` | `.../lib/delta/format` |
| `LenStringMax` reference | `string_format_writer.LenStringMax` | Local `lenRightAlignMax = 17` |
| Import sites updated | — | 6 files across golf, juliett, papa, whiskey |
| Downstream callers | Import `internal/foxtrot/format` | Import `lib/delta/format` |

# catgut

String format read/write interfaces for ring buffer operations.

## Key Interfaces

- `StringFormatReader`: Read formatted strings from ring buffer
- `StringFormatWriter`: Write formatted strings
- `StringFormatReadWriter`: Combined read/write interface

## Known vet warning: `noescape` "possible misuse of unsafe.Pointer"

`go vet` warns on `string.go:61`:

```
lib/bravo/catgut/string.go:61:9: possible misuse of unsafe.Pointer
```

The `noescape` function is the canonical Go-runtime trick for hiding a
pointer from escape analysis (see Go issues 23382 and 7921). The XOR-
with-zero round-trip through `uintptr` is exactly what tripping vet's
unsafeptr analyzer is for — it's an intentional false positive.

**Do not "fix" by removing the round-trip.** The trick is what makes
`(*String).copyCheck` work without forcing the receiver to escape.

The Go runtime accepts the same warning on its own copy of `noescape`;
there is no source-level directive that silences `go vet`'s unsafeptr
analyzer. Tracked in #155.

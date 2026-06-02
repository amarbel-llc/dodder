---
status: exploring
date: 2026-06-02
promotion-criteria:
---

# Type-Defined Field Index

## Problem Statement

Dodder objects carry typed fields --- both metadata and, for typed blobs, the
fields a type's schema declares --- but there is no index over field *values*.
Answering "which object has field `X` equal to `V`?" today requires either a
linear scan of candidate objects or a hand-rolled workaround such as encoding
the value into a derived lookup tag. Several features need exactly this
value-to-object resolution; the first concrete consumer is cutting-garden
receipt ingest (RFC 0003), where a re-capture must find the existing zettel
whose `foreign_key` matches a URL or directory path so the capture lands as a
new *version* rather than an unrelated object.

The gap is a general one: a type should be able to declare which of its fields
are indexable, and the store should maintain a value-keyed index over those
fields incrementally as objects are committed, so lookups are direct rather than
O(n) scans or stringly-typed tag hacks.

## Sketch (to expand)

The intended direction, not yet designed in detail:

- A type's TOML schema (`type_blobs.TomlV2` `fields`) declares which fields
  participate in the index.
- The store maintains a value-keyed index over those fields, updated
  incrementally on commit alongside the existing stream index.
- A query resolves `field = value` to object identity, reusing the existing
  query system where possible.

Candidate resolution mechanisms carried over from RFC 0003's "Future Work:
Foreign-Key Resolution" --- derived lookup tag, linear scan, and a dedicated
field index --- are the starting menu for this design. The field index is the
recommended end state; the others are fallbacks worth recording.

## Related Deferred Work

- **Cutting-garden multi-root fan-out.** A multi-root capture produces one
  receipt for several foreign keys. The eventual mapping --- one zettel per
  root, each referencing the shared receipt --- depends on value-to-object
  resolution and is in scope to revisit here once the index exists. RFC 0003 v1
  skips multi-root receipts with a warning.

## Limitations

This feature is in the `exploring` stage: the problem is defined, but no
interface, persistence format, or incremental-maintenance strategy has been
chosen. Until it lands, consumers use explicit create/update semantics
(RFC 0003 v1) or other per-feature workarounds.

## More Information

- RFC 0003: Cutting-Garden Receipt Ingest --- the motivating consumer; its
  "Future Work: Foreign-Key Resolution" section defers automatic resolution to
  this record.

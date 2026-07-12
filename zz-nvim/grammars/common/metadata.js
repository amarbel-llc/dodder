// Shared rules for the hyphence metadata block (see docs/man.7/hyphence.md).
//
// A metadata block is fenced by two "---" lines. Each metadata line begins with
// a single-character prefix, a space, and content:
//   # text          description
//   - identifier    tag or object reference (whole remainder; may carry a
//                   "< markl" lock inline, matching zz-vim which did not split it)
//   @ markl-id      blob reference (a markl id) ...
//   @ file-path     ... or a file path (base.ext)
//   ! type-string   type, optionally "! type@markl-id" with a lock
//   < object-id     explicit object reference
//   % text          comment
//
// These rules assume the grammar sets `extras: []` -- newlines are significant
// and consumed explicitly, so line boundaries survive into the tree.
//
// Requires the markl rules (markl_id, ...) to be spread into the same grammar.

module.exports = {
  // The metadata block ends exactly at the closing "---"; the fence line's
  // terminating newline (and any following blank line / body) belongs to
  // whatever consumes the metadata, so this composes with both the opaque
  // hyphence body and the structured organize body without a newline conflict.
  metadata: $ =>
    seq(
      '---',
      '\n',
      repeat($._metadata_line),
      '---',
    ),

  _metadata_line: $ =>
    choice(
      $.description_line,
      $.tag_line,
      $.blob_line,
      $.type_line,
      $.ref_line,
      $.comment_line,
    ),

  description_line: $ =>
    seq('#', ' ', field('text', $.description_text), '\n'),
  description_text: $ => token(/[^\n]+/),

  tag_line: $ => seq('-', ' ', field('name', $.tag_name), '\n'),
  tag_name: $ => token(/[^\n]+/),

  ref_line: $ => seq('<', ' ', field('ref', $.reference_id), '\n'),
  reference_id: $ => token(/[^\n]+/),

  type_line: $ =>
    seq(
      '!',
      ' ',
      field('type', $.type_name),
      optional(seq('@', field('lock', $.markl_id))),
      '\n',
    ),
  type_name: $ => token(/[^@\n]+/),

  blob_line: $ =>
    seq('@', ' ', field('ref', choice($.markl_id, $.file_path)), '\n'),
  // A file path must contain a ".", which a markl id never does, so the two
  // are mutually exclusive and longest-match separates them cleanly.
  file_path: $ => token(/[^\n@]*\.[^\n@.]+/),

  comment_line: $ =>
    seq('%', optional(field('text', $.comment_text)), '\n'),
  comment_text: $ => token(/[^\n]+/),
};

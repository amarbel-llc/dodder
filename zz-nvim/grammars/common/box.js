// Shared rules for the box format (see docs/man.7/box.md), dodder's compact
// one-line object representation, used inside organize-text object lines:
//
//   [object-id @blob-digest !type tag1 tag2 key=value] description
//
// Items inside the brackets are space-separated. Object id may be left/right,
// !type, a bare tag, or /repo. Tags prefixed with % are computed/display-only.
// Field values are bare or Go-quoted ("value with \"quotes\"").
//
// Assumes the grammar sets `extras: []`; spaces inside the brackets are matched
// explicitly. Requires the markl rules (markl_id) in the same grammar.

module.exports = {
  box: $ =>
    seq(
      '[',
      repeat(choice($._box_item, $._box_space)),
      ']',
    ),
  _box_space: $ => token(/[ \t]+/),

  _box_item: $ =>
    choice(
      $.box_blob,
      $.box_type,
      $.box_computed_tag,
      $.box_field,
      $.box_object_id,
      $.box_quoted,
      $.box_tag,
    ),

  box_blob: $ => seq('@', $.markl_id),

  box_type: $ => seq('!', field('name', $.box_ident)),

  box_computed_tag: $ => seq('%', field('name', $.box_ident)),

  box_field: $ =>
    seq(field('key', $.box_ident), '=', field('value', $._box_value)),
  _box_value: $ => choice($.box_quoted, $.box_bare_value),
  box_bare_value: $ => token(/[^ \t\]"]+/),

  box_object_id: $ =>
    choice(
      seq(field('left', $.box_ident), '/', field('right', $.box_ident)),
      seq('/', field('name', $.box_ident)),
    ),

  box_tag: $ => $.box_ident,

  box_ident: $ => token(/[a-zA-Z0-9][a-zA-Z0-9_-]*/),

  box_quoted: $ =>
    seq('"', repeat(choice($.box_escape, token.immediate(/[^"\\]+/))), '"'),
  box_escape: $ => token.immediate(/\\./),
};

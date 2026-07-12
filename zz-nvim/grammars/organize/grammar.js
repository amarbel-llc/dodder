// tree-sitter grammar for dodder's organize-text format.
// See docs/man.7/organize-text.md.
//
//   ---
//   ! task
//   - project-alpha
//   ---
//
//   # heading-tag, other-tag
//   - [one/uno !task status=todo] fix the login bug
//   % [two/uno] a virtual-typed object
//
// An optional hyphence metadata header, then a body of markdown-style heading
// lines (# tags, comma-separated) and object lines (prefixed - for a known type
// or % for an unknown/virtual type) in box format with a description trailer.
//
// Reuses the shared hyphence metadata rules (../common/metadata.js), the box
// rules (../common/box.js), and the markl rules (../common/markl.js).

const markl = require('../common/markl.js');
const metadata = require('../common/metadata.js');
const box = require('../common/box.js');

function sepBy1(sep, rule) {
  return seq(rule, repeat(seq(sep, rule)));
}

module.exports = grammar({
  name: 'dodder_organize',

  extras: $ => [],

  // A box_ident may begin a bare tag or the left side of an object id (before
  // "/"); GLR picks the right one via lookahead.
  conflicts: $ => [
    [$.box_object_id, $.box_tag],
  ],

  rules: {
    source_file: $ => seq(optional($.metadata), repeat($._body_line)),

    _body_line: $ => choice($.heading, $.object_line, $.blank_line),

    blank_line: $ => '\n',

    heading: $ =>
      seq(
        field('marker', $.heading_marker),
        ' ',
        sepBy1($._heading_comma, $.heading_tag),
        '\n',
      ),
    heading_marker: $ => token(/#+/),
    _heading_comma: $ => token(/,[ \t]*/),
    heading_tag: $ => token(/[^\n,]+/),

    object_line: $ =>
      seq(
        field('prefix', choice('-', '%')),
        ' ',
        $.box,
        optional(seq(' ', field('description', $.description))),
        '\n',
      ),
    description: $ => token(/[^\n]+/),

    ...metadata,
    ...box,
    ...markl,
  },
});

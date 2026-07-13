// tree-sitter grammar for doddish, dodder's query language.
// See docs/man.7/doddish.md and go/internal/0/doddish/op.go.
//
//   predicate[sigil][genre]
//
// Terms are space-separated (AND). A term is a predicate (bare tag, !type,
// object id, dependent -tag, or empty) optionally followed by one or more
// combinable sigils (: + . ?) and a comma-separated genre list (z t e b k r or
// full names). Terms and groups may be negated with ^; groups use [ ] with
// comma = OR inside; = marks an exact match.
//
// Whitespace is significant (it separates terms), so extras is empty and term
// separators are matched explicitly. Genres attach only to a whole term/group
// (after "]"), never to a group's inner items -- that keeps the group's comma
// separator unambiguous with the genre-list comma.

const { sepBy1 } = require('../common/util.js');

module.exports = grammar({
  name: 'doddish',

  extras: $ => [],

  rules: {
    source_file: $ =>
      seq(
        optional($._sep),
        optional(seq($._element, repeat(seq($._sep, $._element)))),
        optional($._sep),
      ),

    // Terms are AND-separated by whitespace; newlines are tolerated too so a
    // query in a buffer (with its trailing newline) parses cleanly.
    _sep: $ => token(/[ \t\n]+/),

    _element: $ => choice($.term, $.group, $.negation),

    negation: $ => seq('^', choice($.term, $.group)),

    // [ item, item ] with an optional trailing sigil/genre suffix. Inner items
    // are bare terms (no genre list) so the "," here is unambiguous.
    group: $ =>
      seq(
        '[',
        optional(sepBy1(',', $._group_item)),
        ']',
        optional($.sigil),
        optional($.genres),
      ),
    _group_item: $ => choice($._bare_term, $.group_negation),
    group_negation: $ => seq('^', $._bare_term),

    // A full term: predicate/sigil plus an optional genre suffix.
    term: $ => seq($._bare_term, optional($.genres)),

    _bare_term: $ =>
      choice(
        seq($._predicate, optional($.sigil)),
        // empty predicate: bare sigil such as ":" or ":." (with genres via term)
        $.sigil,
      ),

    _predicate: $ =>
      choice(
        $.exact,
        $.type_filter,
        $.object_id,
        $.dependent_tag,
        $.tag,
      ),

    exact: $ => seq('=', choice($.type_filter, $.tag)),

    type_filter: $ => seq('!', field('name', $.identifier)),

    dependent_tag: $ => seq('-', field('name', $.identifier)),

    object_id: $ =>
      choice(
        seq(field('left', $.identifier), '/', field('right', $.identifier)),
        seq('/', field('name', $.identifier)),
      ),

    tag: $ => $.identifier,

    // Combinable sigils: ":", ":.", "+?", etc.
    sigil: $ => repeat1(choice(':', '+', '.', '?')),

    genres: $ => sepBy1(',', $.genre),
    genre: $ => $.identifier,

    identifier: $ => token(/[a-zA-Z0-9][a-zA-Z0-9_-]*/),
  },
});

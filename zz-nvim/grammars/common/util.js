// Shared grammar-construction helpers, not tied to any one dodder format.
//
// grammar.js DSL helpers (seq, repeat, ...) are provided as globals by
// tree-sitter and remain in scope inside required modules.

// One or more `rule`, separated by `sep` (no trailing separator).
function sepBy1(sep, rule) {
  return seq(rule, repeat(seq(sep, rule)));
}

module.exports = { sepBy1 };

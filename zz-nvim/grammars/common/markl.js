// Shared rules for markl-id tokens (see docs/man.7/markl-id.md).
//
// A markl id has the text form:  [purpose@]format-data
//   purpose  optional semantic context, may contain hyphens and underscores,
//            terminated by "@" (e.g. dodder-repo-public_key-v1@)
//   format   the blech32 human-readable part / algorithm (e.g. blake2b256,
//            ed25519_sig, ed25519_pub, sha256) -- letters, digits, underscores
//   data     the blech32 payload + checksum, from the alphabet
//            qpzry9x8gf2tvdw0s3jn54khce6mua7l (a superset of [a-z0-9] minus
//            1 b i o -- matched leniently here as [a-z0-9]+ for highlighting)
//
// grammar.js DSL helpers (seq, field, token, prec, optional, ...) are provided
// as globals by tree-sitter and remain in scope inside required modules.

module.exports = {
  markl_id: $ =>
    seq(
      optional($.markl_purpose),
      field('format', $.markl_format),
      '-',
      field('data', $.markl_data),
    ),

  // The trailing "@" is part of the token so the lexer can distinguish a
  // purpose (which requires "@") from a bare format without conflict.
  markl_purpose: $ => token(/[a-z0-9_][a-z0-9_-]*@/),

  markl_format: $ => token(/[a-z0-9_]+/),

  markl_data: $ => token(/[a-z0-9]+/),
};

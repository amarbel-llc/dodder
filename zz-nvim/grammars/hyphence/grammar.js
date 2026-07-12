// tree-sitter grammar for the hyphence (hyphen-fence) serialization format,
// dodder's primary object edit format. See docs/man.7/hyphence.md.
//
//   ---
//   # description
//   - tag
//   ! type
//   ---
//                <- required blank line
//   body content (language resolved per object type; injected)
//
// The metadata block and its line rules are shared from ../common/metadata.js;
// the markl-id token rules from ../common/markl.js. The (body) node is left
// opaque here and reparsed in a sub-language via queries/hyphence/injections.scm.

const markl = require('../common/markl.js');
const metadata = require('../common/metadata.js');

module.exports = grammar({
  name: 'hyphence',

  // Newlines are significant (line-oriented metadata); no implicit whitespace.
  extras: $ => [],

  rules: {
    source_file: $ => seq($.metadata, optional($.body)),

    // Everything after the metadata's closing fence: the fence line's newline,
    // the required blank line (hence the leading \n\n), then the remaining
    // content. Opaque here, injected downstream. Requiring two newlines means a
    // metadata-only document that merely ends in a newline produces no body.
    // NB: use (.|\n)* rather than [\s\S]* -- tree-sitter's regex does not
    // extend a [\s\S] character class across newlines, which truncates the body.
    body: $ => token(/\n\n(.|\n)*/),

    ...metadata,
    ...markl,
  },
});

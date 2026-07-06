package main

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
)

// seedType is one row of the dodder.net seed-set table (FDR-0010 Phase 3):
// the 57 generic file-format types triaged as class (a) SEED-SET in
// docs/plans/2026-07-04-type-reconciliation-groups.md §"Group 3 triage
// results". Blobs are GENERATED from this table (name → extension / binary /
// vim-syntax), not exported from the user's repo; formatter-rich types
// (plantuml, graphviz, ly, webp) deliberately ship TRIVIAL blobs — blob-backed
// tool pipelines are future per-type work (group3-decisions-2026-07-05.md §2).
type seedType struct {
	// Name is the type object id without the "!" prefix; it doubles as the
	// output filename stem (<name>.type).
	Name          string
	Description   string
	FileExtension string
	Binary        bool
	// VimSyntaxType uses vim's REAL filetype names (":help filetype"), not
	// the extension (javascript, not js; sh, not bash). Empty for binary
	// formats, where a syntax hint is meaningless.
	VimSyntaxType string
	// NixpkgsFormatterCandidates records DETERMINISTIC text-rendering tools
	// available in nixpkgs that could back a future text formatter for this
	// type. Analysis only (group3-decisions-2026-07-05.md §1) — the seed
	// blobs stay trivial; this is emitted as a TOML comment, never as an
	// actual formatter. Determinism caveats are inlined in parentheses.
	// Empty means no suitable deterministic candidate.
	NixpkgsFormatterCandidates string
}

// Shared candidate strings. exiftool's default output includes the File*
// group (filesystem timestamps, inode change dates), which varies per
// checkout — a deterministic formatter must exclude it.
const (
	nixExiftool    = "exiftool (exclude File* filesystem-timestamp tags for determinism)"
	nixUnzipList   = "unzip -l (structure listing)"
	nixXmllint     = "xmllint --format (libxml2)"
	nixGpgPackets  = "gpg --list-packets (gnupg)"
	nixFontMetrics = "otfinfo -i (lcdf-typetools); fc-scan (fontconfig)"
)

// seedTypes is the authoritative seed-set table, sorted by Name (asserted in
// main_test.go). Binary flags follow the user repo's triage blob notes: the
// shared `binary = true` blob families stay binary even for text-ish formats
// (ics, vcf, mht, ps, rtf, asc); eml — blobless in the user repo — is
// generated as text since RFC-822 mail is unambiguously text.
var seedTypes = []seedType{
	{
		Name:                       "aax",
		Description:                "audible audiobook (drm)",
		FileExtension:              "aax",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "asc",
		Description:                "ascii-armored pgp data",
		FileExtension:              "asc",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixGpgPackets,
	},
	{
		Name:                       "awk",
		Description:                "awk script",
		FileExtension:              "awk",
		VimSyntaxType:              "awk",
		NixpkgsFormatterCandidates: "gawk -o- (pretty-print)",
	},
	{
		Name:                       "bash",
		Description:                "bash shell script",
		FileExtension:              "bash",
		VimSyntaxType:              "sh",
		NixpkgsFormatterCandidates: "shfmt",
	},
	{
		Name:                       "caf",
		Description:                "apple core audio format",
		FileExtension:              "caf",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:          "cfg",
		Description:   "generic configuration file",
		FileExtension: "cfg",
		VimSyntaxType: "cfg",
	},
	{
		Name:                       "csv",
		Description:                "comma-separated values",
		FileExtension:              "csv",
		VimSyntaxType:              "csv",
		NixpkgsFormatterCandidates: "mlr --icsv --opprint cat (miller); csvlook (csvkit)",
	},
	{
		Name:                       "dcm",
		Description:                "dicom medical image",
		FileExtension:              "dcm",
		Binary:                     true,
		NixpkgsFormatterCandidates: "dcmdump (dcmtk); " + nixExiftool,
	},
	{
		Name:                       "dmg",
		Description:                "apple disk image",
		FileExtension:              "dmg",
		Binary:                     true,
		NixpkgsFormatterCandidates: "7z l (p7zip)",
	},
	{
		Name:                       "doc",
		Description:                "microsoft word legacy document",
		FileExtension:              "doc",
		Binary:                     true,
		NixpkgsFormatterCandidates: "catdoc; antiword",
	},
	{
		// Description carried from the user's repo (triage table).
		Name:          "eml",
		Description:   "archive html email",
		FileExtension: "eml",
		VimSyntaxType: "mail",
	},
	{
		Name:                       "excalidraw",
		Description:                "excalidraw sketch (json)",
		FileExtension:              "excalidraw",
		VimSyntaxType:              "json",
		NixpkgsFormatterCandidates: "jq .",
	},
	{
		Name:                       "gif",
		Description:                "graphics interchange format image",
		FileExtension:              "gif",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "gpg",
		Description:                "openpgp binary data",
		FileExtension:              "gpg",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixGpgPackets,
	},
	{
		Name:                       "graphviz",
		Description:                "graphviz dot graph",
		FileExtension:              "dot",
		VimSyntaxType:              "dot",
		NixpkgsFormatterCandidates: "dot -Tplain (graphviz; deterministic for a given engine version)",
	},
	{
		Name:                       "gz",
		Description:                "gzip-compressed data",
		FileExtension:              "gz",
		Binary:                     true,
		NixpkgsFormatterCandidates: "zcat (gzip)",
	},
	{
		Name:                       "heic",
		Description:                "high efficiency image container",
		FileExtension:              "heic",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		// Text format, but kept binary to match the user repo's shared
		// `binary = true` blob; no core vim filetype for iCalendar.
		Name:          "ics",
		Description:   "icalendar calendar data",
		FileExtension: "ics",
		Binary:        true,
	},
	{
		Name:          "indd",
		Description:   "adobe indesign document",
		FileExtension: "indd",
		Binary:        true,
	},
	{
		Name:                       "iso",
		Description:                "optical disc image",
		FileExtension:              "iso",
		Binary:                     true,
		NixpkgsFormatterCandidates: "isoinfo -l (cdrkit)",
	},
	{
		// Duplicate of !jpg (which carries the usage); triage recommends
		// canonicalizing on !jpg, but both are in the seed set as-is.
		Name:                       "jpeg",
		Description:                "jpeg image",
		FileExtension:              "jpeg",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "jpg",
		Description:                "jpeg image",
		FileExtension:              "jpg",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:          "jq",
		Description:   "jq filter script",
		FileExtension: "jq",
		VimSyntaxType: "jq",
	},
	{
		Name:                       "js",
		Description:                "javascript source",
		FileExtension:              "js",
		VimSyntaxType:              "javascript",
		NixpkgsFormatterCandidates: "prettier",
	},
	{
		// Password-gated: keepassxc-cli needs credentials, so no
		// unattended deterministic formatter exists.
		Name:          "kdbx",
		Description:   "keepass password database",
		FileExtension: "kdbx",
		Binary:        true,
	},
	{
		Name:                       "kmz",
		Description:                "google earth kmz (zipped kml)",
		FileExtension:              "kmz",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixUnzipList,
	},
	{
		// No deterministic candidate: lilypond embeds its version in
		// pdf/midi output.
		Name:          "ly",
		Description:   "lilypond music notation",
		FileExtension: "ly",
		VimSyntaxType: "lilypond",
	},
	{
		Name:                       "m4a",
		Description:                "mpeg-4 audio",
		FileExtension:              "m4a",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "m4b",
		Description:                "mpeg-4 audiobook",
		FileExtension:              "m4b",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "m4r",
		Description:                "mpeg-4 ringtone",
		FileExtension:              "m4r",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:          "mht",
		Description:   "mhtml web archive",
		FileExtension: "mht",
		Binary:        true,
	},
	{
		Name:                       "mobi",
		Description:                "mobipocket ebook",
		FileExtension:              "mobi",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "mov",
		Description:                "quicktime video",
		FileExtension:              "mov",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "mp3",
		Description:                "mpeg audio layer iii",
		FileExtension:              "mp3",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "odg",
		Description:                "opendocument graphics",
		FileExtension:              "odg",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixUnzipList,
	},
	{
		Name:                       "ods",
		Description:                "opendocument spreadsheet",
		FileExtension:              "ods",
		Binary:                     true,
		NixpkgsFormatterCandidates: "ssconvert (gnumeric)",
	},
	{
		Name:                       "odt",
		Description:                "opendocument text",
		FileExtension:              "odt",
		Binary:                     true,
		NixpkgsFormatterCandidates: "pandoc -f odt -t markdown",
	},
	{
		Name:                       "otf",
		Description:                "opentype font",
		FileExtension:              "otf",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixFontMetrics,
	},
	{
		Name:                       "pages",
		Description:                "apple pages document",
		FileExtension:              "pages",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixUnzipList,
	},
	{
		Name:                       "pdf",
		Description:                "portable document format",
		FileExtension:              "pdf",
		Binary:                     true,
		NixpkgsFormatterCandidates: "pdftotext (poppler-utils); " + nixExiftool,
	},
	{
		Name:                       "plantuml",
		Description:                "plantuml diagram source",
		FileExtension:              "puml",
		VimSyntaxType:              "plantuml",
		NixpkgsFormatterCandidates: "plantuml -tutxt (ascii-art render; layout varies across versions)",
	},
	{
		// Description carried from the user's repo (triage table).
		Name:                       "png",
		Description:                "portable network graphics format",
		FileExtension:              "png",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "pptx",
		Description:                "microsoft powerpoint presentation",
		FileExtension:              "pptx",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixUnzipList,
	},
	{
		Name:                       "ps",
		Description:                "postscript document",
		FileExtension:              "ps",
		Binary:                     true,
		NixpkgsFormatterCandidates: "ps2ascii (ghostscript)",
	},
	{
		Name:          "pxd",
		Description:   "pixelmator document",
		FileExtension: "pxd",
		Binary:        true,
	},
	{
		Name:                       "rego",
		Description:                "open policy agent rego policy",
		FileExtension:              "rego",
		VimSyntaxType:              "rego",
		NixpkgsFormatterCandidates: "opa fmt (open-policy-agent)",
	},
	{
		Name:                       "rtf",
		Description:                "rich text format document",
		FileExtension:              "rtf",
		Binary:                     true,
		NixpkgsFormatterCandidates: "pandoc -f rtf -t markdown",
	},
	{
		Name:                       "svg",
		Description:                "scalable vector graphics image",
		FileExtension:              "svg",
		VimSyntaxType:              "svg",
		NixpkgsFormatterCandidates: nixXmllint,
	},
	{
		Name:                       "tiff",
		Description:                "tagged image file format",
		FileExtension:              "tiff",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "ttf",
		Description:                "truetype font",
		FileExtension:              "ttf",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixFontMetrics,
	},
	{
		// Text format, but kept binary to match the user repo's shared
		// `binary = true` blob; no core vim filetype for vCard.
		Name:          "vcf",
		Description:   "vcard contact data",
		FileExtension: "vcf",
		Binary:        true,
	},
	{
		Name:                       "wav",
		Description:                "waveform audio",
		FileExtension:              "wav",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "webp",
		Description:                "webp image",
		FileExtension:              "webp",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool + "; dwebp -o - (libwebp; decode to png)",
	},
	{
		Name:                       "xcf",
		Description:                "gimp image",
		FileExtension:              "xcf",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixExiftool,
	},
	{
		Name:                       "xlsx",
		Description:                "microsoft excel spreadsheet",
		FileExtension:              "xlsx",
		Binary:                     true,
		NixpkgsFormatterCandidates: "xlsx2csv",
	},
	{
		Name:                       "xml",
		Description:                "extensible markup language document",
		FileExtension:              "xml",
		VimSyntaxType:              "xml",
		NixpkgsFormatterCandidates: nixXmllint,
	},
	{
		Name:                       "zip",
		Description:                "zip archive",
		FileExtension:              "zip",
		Binary:                     true,
		NixpkgsFormatterCandidates: nixUnzipList,
	},
}

// render produces the hyphence .type file for the entry: a metadata section
// (description + blob type) followed by the TomlV2 TOML blob body, ready for
// a future `dodder checkin` / import into a seed repo. The blank line between
// the closing boundary and the blob body is load-bearing: without it the
// hyphence parser silently drops the blob (issue #41).
func (entry seedType) render() []byte {
	var sb strings.Builder

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "# %s\n", entry.Description)
	fmt.Fprintf(&sb, "! %s\n", strings.TrimPrefix(ids.TypeTomlTypeV2, "!"))
	sb.WriteString("---\n")
	sb.WriteString("\n")

	candidates := entry.NixpkgsFormatterCandidates

	if candidates == "" {
		candidates = "none"
	}

	fmt.Fprintf(&sb, "# nixpkgs-formatter-candidates: %s\n", candidates)
	fmt.Fprintf(&sb, "binary = %t\n", entry.Binary)
	fmt.Fprintf(&sb, "file-extension = %q\n", entry.FileExtension)

	if entry.VimSyntaxType != "" {
		fmt.Fprintf(&sb, "vim-syntax-type = %q\n", entry.VimSyntaxType)
	}

	return []byte(sb.String())
}

//go:build test

package repo_configs

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Pins that the bare V2 config coder decodes a body-only TOML blob.
// Config blobs are stored and presented bare: the leading hyphence
// `---` / `! toml-config-v2` frame is object-level metadata that
// hyphence's CoderToTypedBlob.DecodeFrom (via readMetadataFrom) strips
// before the body ever reaches this bare Coder.Blob / DecodeV2 decoder.
// Both production call sites feed body-only bytes —
// november/store_config/persist.go (bootstrap mutable-config read of the
// blob content) and uniform/commands_dodder/konfig_edit.go (the temp
// file is a verbatim copy of the bare blob bytes) — so a framed blob
// never reaches DecodeV2.
//
// This replaces the former TestDecodeV2_LenientWithHyphenceWrapper,
// which asserted tommy leniently swallowed an embedded `---` frame. That
// path does not exist in production, and tommy v0.4.3's decode-
// normalization made the parser strict about a leading fence, so the
// wrapped case was skipped and is now removed. See #257.
func TestDecodeV2_DecodesBareBody(t1 *testing.T) {
	t := ui.MakeT(t1)

	body := strings.Join([]string{
		`blob-stores = [".default"]`,
		"",
		"[defaults]",
		`type = "!md"`,
		"",
	}, "\n")

	doc, err := DecodeV2([]byte(body))
	if err != nil {
		t.Fatalf("bare TOML failed to decode: %v", err)
	}

	data := doc.Data()

	if len(data.BlobStores) != 1 {
		t.Fatalf("expected exactly 1 blob-store, got %d", len(data.BlobStores))
	}

	if data.Defaults.Type.String() != "!md" {
		t.Fatalf("expected defaults.type !md, got %q", data.Defaults.Type)
	}
}

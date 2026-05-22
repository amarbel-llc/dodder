//go:build test

package repo_configs

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Pins Tommy's lenient TOML parsing of leading hyphence-fence lines
// (`---`) and type-directive lines (`! toml-config-v2`). Older stores
// hold konfig blobs that begin with that wrapper; the load path
// decodes them via the bare `Coder.Blob` decoder and relies on Tommy
// to ignore the wrapper lines. If Tommy ever becomes strict, every
// such store would fail to load.
func TestDecodeV2_LenientWithHyphenceWrapper(t1 *testing.T) {
	t := ui.MakeT(t1)

	body := strings.Join([]string{
		`blob-stores = [".default"]`,
		"",
		"[defaults]",
		`type = "!md"`,
		"",
	}, "\n")

	wrapped := strings.Join([]string{
		"---",
		"! toml-config-v2",
		"---",
		"",
		body,
	}, "\n")

	docBare, err := DecodeV2([]byte(body))
	if err != nil {
		t.Fatalf("bare TOML failed to decode: %v", err)
	}
	bare := docBare.Data()

	docWrapped, err := DecodeV2([]byte(wrapped))
	if err != nil {
		t.Fatalf(
			"wrapped TOML failed to decode — v14 stores would "+
				"hard-fail on load. Tommy parser became strict? "+
				"error: %v", err,
		)
	}
	wrap := docWrapped.Data()

	// Same blob-stores set on both forms.
	if len(bare.BlobStores) != len(wrap.BlobStores) {
		t.Fatalf(
			"blob-stores count mismatch: bare=%d wrapped=%d",
			len(bare.BlobStores), len(wrap.BlobStores),
		)
	}
	if len(bare.BlobStores) != 1 {
		t.Fatalf("expected exactly 1 blob-store in bare form, got %d", len(bare.BlobStores))
	}

	// Same defaults.type on both forms.
	if bare.Defaults.Type.String() != wrap.Defaults.Type.String() {
		t.Fatalf(
			"defaults.type mismatch: bare=%q wrapped=%q",
			bare.Defaults.Type, wrap.Defaults.Type,
		)
	}
}

//go:build test

package repo_configs

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Pins that the bare V3 config coder decodes a body-only TOML blob,
// including the new default-blob-store field (amarbel-llc/dodder#223,
// FDR-0016 D1). Mirrors TestDecodeV2_DecodesBareBody's bare-body
// assumption: production feeds body-only bytes, never a hyphence-framed
// blob, to this decoder.
func TestDecodeV3_DecodesBareBody(t1 *testing.T) {
	t := ui.MakeT(t1)

	body := strings.Join([]string{
		`blob-stores = [".default"]`,
		`default-blob-store = "default@blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d"`,
		"",
		"[defaults]",
		`type = "!md"`,
		"",
	}, "\n")

	doc, err := DecodeV3([]byte(body))
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

	if !data.DefaultBlobStore.HasDigest() {
		t.Fatalf("expected default-blob-store to carry a digest, got %q", data.DefaultBlobStore.Canonical())
	}
}

// Round-trip: encode a V3 value, decode it back, confirm the
// default-blob-store field survives byte-identical.
func TestV3_RoundTripsDefaultBlobStore(t1 *testing.T) {
	t := ui.MakeT(t1)

	body := strings.Join([]string{
		`blob-stores = ["default"]`,
		`default-blob-store = "default@blake2b256-zcfmrghzp36r4r4qxtrh4t8xcd5g0f3mkpm8f3swac0vr5x503msyfsu3d"`,
		"",
		"[defaults]",
		`type = "!md"`,
		"",
	}, "\n")

	decoded, err := DecodeV3([]byte(body))
	if err != nil {
		t.Fatalf("failed to decode fixture body: %v", err)
	}

	encoded, err := decoded.Encode()
	if err != nil {
		t.Fatalf("failed to encode V3: %v", err)
	}

	redecoded, err := DecodeV3(encoded)
	if err != nil {
		t.Fatalf("failed to decode re-encoded V3: %v", err)
	}

	expected := decoded.Data().DefaultBlobStore.Canonical()
	actual := redecoded.Data().DefaultBlobStore.Canonical()

	if expected != actual {
		t.Fatalf("default-blob-store did not round-trip: expected %q, got %q", expected, actual)
	}
}

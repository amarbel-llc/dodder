package repo_identity

import (
	"testing"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"

	// Core format registrations (ed25519_pub et al.) so Set of a bare
	// format-data id resolves. Registers piggy-* purposes only — the
	// dodder-* purposes deliberately stay unregistered here (see the
	// StringWithFormat comment below).
	_ "github.com/amarbel-llc/piggy/go/pkgs/markl_registrations"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// A real ed25519 public key (bare `ed25519_pub-...` format-data form), lifted
// from the zz-tests_bats import fixtures. blech32 carries a checksum, so this
// must be a genuine value, not an arbitrary string.
const validPubkey = "ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6"

func TestRender(t1 *testing.T) {
	t := ui.MakeT(t1)

	var pubkey markl.Id
	if err := pubkey.Set(validPubkey); err != nil {
		t.Fatalf("Set(%q): %s", validPubkey, err)
	}

	// Render must use the bare String() (format-data) form, NOT
	// StringWithFormat(). A real repo pubkey carries the
	// `dodder-repo-public_key-v1` purpose, so StringWithFormat() would emit
	// `dodder-repo-public_key-v1@ed25519_pub-...` and Render would produce a
	// double-`@`. That purposed case can't be reproduced in this isolated
	// package (it doesn't register dodder markl purposes, so Set of a purposed
	// id panics "no purpose registered"); it is guarded end-to-end by the
	// info-repo id BATS test (asserts `^<handle>@ed25519_pub-`). Here we pin
	// the contract that Render emits the bare String() form.
	bare := pubkey.String()

	// The empty-pubkey case relies on a freshly-zeroed markl.Id reporting
	// IsNull(); assert that assumption holds before depending on it.
	var nullPubkey markl.Id
	if !nullPubkey.IsNull() {
		t.Fatalf("zero markl.Id IsNull() = false, want true")
	}

	for _, testCase := range []struct {
		name   string
		handle string
		pubkey mad_domain_interfaces.MarklId
		want   string
	}{
		{
			name:   "handle and bare pubkey joined with @ (pubkey purpose dropped)",
			handle: "default",
			pubkey: pubkey,
			want:   "default@" + bare,
		},
		{
			name:   "empty handle drops the leading @",
			handle: "",
			pubkey: pubkey,
			want:   bare,
		},
		{
			name:   "null pubkey returns handle unchanged",
			handle: "default",
			pubkey: nullPubkey,
			want:   "default",
		},
		{
			name:   "nil pubkey returns handle unchanged",
			handle: "default",
			pubkey: nil,
			want:   "default",
		},
	} {
		if got := Render(testCase.handle, testCase.pubkey); got != testCase.want {
			t.Errorf(
				"Render(%q, %s) = %q, want %q",
				testCase.handle,
				testCase.name,
				got,
				testCase.want,
			)
		}
	}
}

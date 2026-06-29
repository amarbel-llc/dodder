package repo_identity

import (
	"testing"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// A real ed25519 public key in StringWithFormat (`ed25519_pub-...`) form,
// lifted from the zz-tests_bats import fixtures. blech32 carries a checksum,
// so this must be a genuine value, not an arbitrary string.
const validPubkey = "ed25519_pub-vhhh5p6qfc9q5fpqm2xmjmetgnagmjpxxqlwlac4uvrhrvjvgevsv5z5q6"

func TestRender(t1 *testing.T) {
	t := ui.MakeT(t1)

	var pubkey markl.Id
	if err := pubkey.Set(validPubkey); err != nil {
		t.Fatalf("Set(%q): %s", validPubkey, err)
	}

	formattedPubkey := pubkey.StringWithFormat()

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
			name:   "handle and pubkey joined with @",
			handle: "default",
			pubkey: pubkey,
			want:   "default@" + formattedPubkey,
		},
		{
			name:   "empty handle drops the leading @",
			handle: "",
			pubkey: pubkey,
			want:   formattedPubkey,
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

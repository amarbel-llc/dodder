package repo_identity

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
)

// Render returns the repo's human-facing identity in the form
// `<handle>@<pubkey>`, e.g. `default@ed25519_pub-9ft3...`. This is a
// display-only string concatenation; it is NOT a registered markl purpose.
//
//   - normal:           handle + "@" + pubkey.StringWithFormat()
//   - empty handle:      pubkey.StringWithFormat() (no leading "@")
//   - nil / null pubkey: handle unchanged
func Render(handle string, pubkey mad_domain_interfaces.MarklId) string {
	if pubkey == nil || pubkey.IsNull() {
		return handle
	}

	formattedPubkey := pubkey.StringWithFormat()

	if handle == "" {
		return formattedPubkey
	}

	return handle + "@" + formattedPubkey
}

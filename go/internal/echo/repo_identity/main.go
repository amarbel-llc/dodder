package repo_identity

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
)

// Render returns the repo's human-facing identity in the form
// `<handle>@<pubkey>`, e.g. `default@ed25519_pub-9ft3...`. This is a
// display-only string in the markl-id `purpose@format-data` shape, with the
// location handle occupying the purpose slot.
//
// The pubkey is rendered via String() (the bare `ed25519_pub-...` format-data
// form), NOT StringWithFormat(): the repo pubkey carries its own registered
// purpose (`dodder-repo-public_key-v1`), so StringWithFormat() would yield
// `dodder-repo-public_key-v1@ed25519_pub-...` and produce a confusing
// double-`@` (`handle@dodder-repo-public_key-v1@ed25519_pub-...`). Dropping the
// pubkey's own purpose and putting the handle in the purpose slot keeps the
// single-`@` shape the design specifies.
//
//   - normal:           handle + "@" + pubkey.String()
//   - empty handle:      pubkey.String() (no leading "@")
//   - nil / null pubkey: handle unchanged
func Render(handle string, pubkey mad_domain_interfaces.MarklId) string {
	if pubkey == nil || pubkey.IsNull() {
		return handle
	}

	barePubkey := pubkey.String()

	if handle == "" {
		return barePubkey
	}

	return handle + "@" + barePubkey
}

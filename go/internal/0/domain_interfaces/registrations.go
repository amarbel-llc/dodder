package domain_interfaces

// Side-effect blank import: triggers init() in madder's
// markl_registrations, which calls markl.RegisterPurpose for every
// purpose dodder uses. After the markl-fork drop, dodder consumes
// madder/pkgs/markl (types only, no init), so the registrations
// have to be triggered explicitly. This package is the lowest
// dodder-internal tier that every downstream caller transitively
// imports — the most uniform place to keep production binaries
// and unit-test binaries in sync.
import (
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"

	"github.com/amarbel-llc/madder/go/pkgs/markl"
)

// PurposeObjectMotherDigestV1 marks a mother slot holding a plain
// digest instead of a signature. Config log entries chain by the
// previous entry's object digest (FDR 0020) — the purpose TYPE stays
// PurposeTypeObjectMotherSig so box encode/decode routes the value
// into the mother slot, while the digest FormatIds permit unsigned
// chains.
const PurposeObjectMotherDigestV1 = "dodder-object-mother-digest-v1"

func init() {
	markl.RegisterPurpose(markl.RegisterPurposeOpts{
		Id:   PurposeObjectMotherDigestV1,
		Type: markl.PurposeTypeObjectMotherSig,
		FormatIds: []string{
			markl.FormatIdHashSha256,
			markl.FormatIdHashBlake2b256,
		},
	})
}

package store_version

import (
	"strconv"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/values"
)

var (
	VNull = Version(values.Int(0))

	// TODO drop support for versions above
	// TODO use golang generation for versions
	V10 = Version(values.Int(10))
	V11 = Version(values.Int(11))
	V12 = Version(values.Int(12))
	V13 = Version(values.Int(13))
	V14 = Version(values.Int(14))

	VCurrent = V13
	VNext    = V14
)

// TODO replace with Int
type Version values.Int

type Getter interface {
	GetStoreVersion() Version
}

func Equals(
	a domain_interfaces.StoreVersion,
	others ...domain_interfaces.StoreVersion,
) bool {
	for _, other := range others {
		if a.GetInt() == other.GetInt() {
			return true
		}
	}

	return false
}

func Less(a, b domain_interfaces.StoreVersion) bool {
	return a.GetInt() < b.GetInt()
}

func LessOrEqual(a, b domain_interfaces.StoreVersion) bool {
	return a.GetInt() <= b.GetInt()
}

func Greater(a, b domain_interfaces.StoreVersion) bool {
	return a.GetInt() > b.GetInt()
}

func GreaterOrEqual(a, b domain_interfaces.StoreVersion) bool {
	return a.GetInt() >= b.GetInt()
}

func (version Version) Less(b domain_interfaces.StoreVersion) bool {
	return Less(version, b)
}

func (version Version) LessOrEqual(b domain_interfaces.StoreVersion) bool {
	return LessOrEqual(version, b)
}

func (version Version) String() string {
	return values.Int(version).String()
}

func (version Version) GetInt() int {
	return values.Int(version).Int()
}

func (version *Version) Set(p string) (err error) {
	var i uint64

	if i, err = strconv.ParseUint(p, 10, 16); err != nil {
		err = errors.Wrap(err)
		return err
	}

	*version = Version(i)

	if VCurrent.Less(version) {
		err = errors.Wrap(ErrFutureStoreVersion{StoreVersion: version})
		return err
	}

	return err
}

func IsVersionLessOrEqualToV11(other domain_interfaces.StoreVersion) bool {
	return LessOrEqual(other, V10)
}

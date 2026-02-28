package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/markl_age_id"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

type V0Common struct {
	StoreVersion      store_version.Version
	Recipients        []string
	CompressionType   compression_type.CompressionType
	LockInternalFiles bool
}

type V0Private struct {
	V0Common
}

var _ ConfigPrivate = &V0Private{}

type V0Public struct {
	V0Common
}

var _ ConfigPublic = &V0Public{}

var _ interfaces.CommandComponentWriter = (*V0Private)(nil)

func (config *V0Common) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	panic(errors.Err405MethodNotAllowed)
}

func (config *V0Private) GetGenesisConfig() ConfigPrivate {
	return config
}

func (config *V0Private) GetGenesisConfigPublic() ConfigPublic {
	return &V0Public{
		V0Common: config.V0Common,
	}
}

func (config *V0Public) GetGenesisConfig() ConfigPublic {
	return config
}

func (config *V0Common) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	return &blob_store_configs.TomlV0{
		AgeEncryption:   *config.GetAgeEncryption(),
		CompressionType: config.CompressionType,
	}
}

func (config V0Common) GetStoreVersion() store_version.Version {
	return config.StoreVersion
}

func (config V0Common) GetPrivateKey() domain_interfaces.MarklId {
	panic(errors.Err405MethodNotAllowed)
}

func (config *V0Common) GetPrivateKeyMutable() domain_interfaces.MarklIdMutable {
	panic(errors.Err405MethodNotAllowed)
}

func (config V0Common) GetPublicKey() domain_interfaces.MarklId {
	panic(errors.Err405MethodNotAllowed)
}

func (config V0Common) GetRepoId() ids.RepoId {
	return ids.RepoId{}
}

func (config *V0Common) GetAgeEncryption() *markl_age_id.Id {
	return &markl_age_id.Id{}
}

func (config *V0Common) GetBlobCompression() interfaces.CLIFlagIOWrapper {
	return &config.CompressionType
}

func (config *V0Common) GetBlobEncryption() interfaces.IOWrapper {
	var ioWrapper interfaces.IOWrapper = ohio.NopeIOWrapper{}
	encryption := config.GetAgeEncryption()

	if encryption != nil {
		var err error
		ioWrapper, err = encryption.GetIOWrapper()
		errors.PanicIfError(err)
	}

	return ioWrapper
}

func (config V0Common) GetLockInternalFiles() bool {
	return config.LockInternalFiles
}

func (config V0Common) GetInventoryListTypeId() string {
	return ids.TypeInventoryListV0
}

func (config V0Common) GetObjectSigMarklTypeId() string {
	return markl.PurposeObjectSigV0
}

func (config V0Common) SetInventoryListTypeString(string) {
	panic(errors.Err405MethodNotAllowed)
}

func (config V0Common) SetObjectSigTypeString(string) {
	panic(errors.Err405MethodNotAllowed)
}

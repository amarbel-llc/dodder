package genesis_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/internal/charlie/store_version"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl_age_id"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
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

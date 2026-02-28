package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
)

// TODO move into own package

func MakeConfig(
	hashFormat domain_interfaces.FormatHash,
	funcJoin func(string, ...string) string,
	compression interfaces.IOWrapper,
	encryption domain_interfaces.MarklId,
) Config {
	var ioWrapper interfaces.IOWrapper = defaultEncryptionIOWrapper

	if encryption != nil {
		var err error
		ioWrapper, err = encryption.GetIOWrapper()
		errors.PanicIfError(err)
	}

	return Config{
		hashFormat:  hashFormat,
		funcJoin:    funcJoin,
		compression: compression,
		encryption:  ioWrapper,
	}
}

var (
	defaultCompressionTypeValue = compression_type.CompressionTypeNone
	defaultEncryptionIOWrapper  = ohio.NopeIOWrapper{}
	DefaultConfig               = Config{
		hashFormat:  blob_store_configs.DefaultHashType,
		compression: &defaultCompressionTypeValue,
		encryption:  &defaultEncryptionIOWrapper,
	}
)

type Config struct {
	hashFormat domain_interfaces.FormatHash
	// TODO replace with path generator interface
	funcJoin    func(string, ...string) string
	compression interfaces.IOWrapper
	encryption  interfaces.IOWrapper
}

func (config Config) GetBlobCompression() interfaces.IOWrapper {
	if config.compression == nil {
		return &defaultCompressionTypeValue
	} else {
		return config.compression
	}
}

func (config Config) GetBlobEncryption() interfaces.IOWrapper {
	if config.encryption == nil {
		return defaultEncryptionIOWrapper
	} else {
		return config.encryption
	}
}

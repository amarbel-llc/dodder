package blob_store_configs

import (
	"fmt"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
)

// ConfigKeyValues returns a map of TOML-tag-named keys to string-formatted
// values for the given config. The set of keys depends on which interfaces the
// config implements.
func ConfigKeyValues(config Config) map[string]string {
	keyValues := make(map[string]string)

	keyValues["blob-store-type"] = config.GetBlobStoreType()

	if configHashType, ok := config.(ConfigHashType); ok {
		keyValues["hash_type-id"] = configHashType.GetDefaultHashTypeId()
		keyValues["supports-multi-hash"] = fmt.Sprint(
			configHashType.SupportsMultiHash(),
		)
	}

	if blobIOWrapper, ok := config.(domain_interfaces.BlobIOWrapper); ok {
		keyValues["encryption"] = blobIOWrapper.GetBlobEncryption().
			StringWithFormat()
		keyValues["compression-type"] = fmt.Sprint(
			blobIOWrapper.GetBlobCompression(),
		)
	}

	if configLocal, ok := config.(ConfigLocalHashBucketed); ok {
		keyValues["hash_buckets"] = fmt.Sprint(
			configLocal.GetHashBuckets(),
		)
		keyValues["lock-internal-files"] = fmt.Sprint(
			configLocal.GetLockInternalFiles(),
		)
	}

	if configArchive, ok := config.(ConfigInventoryArchive); ok {
		keyValues["loose-blob-store-id"] = configArchive.
			GetLooseBlobStoreId().String()
		keyValues["max-pack-size"] = fmt.Sprint(
			configArchive.GetMaxPackSize(),
		)
	}

	if configDelta, ok := config.(DeltaConfigImmutable); ok {
		keyValues["delta.enabled"] = fmt.Sprint(
			configDelta.GetDeltaEnabled(),
		)
		keyValues["delta.algorithm"] = configDelta.GetDeltaAlgorithm()
		keyValues["delta.min-blob-size"] = fmt.Sprint(
			configDelta.GetDeltaMinBlobSize(),
		)
		keyValues["delta.max-blob-size"] = fmt.Sprint(
			configDelta.GetDeltaMaxBlobSize(),
		)
		keyValues["delta.size-ratio"] = fmt.Sprint(
			configDelta.GetDeltaSizeRatio(),
		)
	}

	if configSFTP, ok := config.(ConfigSFTPConfigExplicit); ok {
		keyValues["host"] = configSFTP.GetHost()
		keyValues["port"] = fmt.Sprint(configSFTP.GetPort())
		keyValues["user"] = configSFTP.GetUser()
		keyValues["private-key-path"] = configSFTP.GetPrivateKeyPath()
		keyValues["remote-path"] = configSFTP.GetRemotePath()
	} else if configSFTPRemote, ok := config.(ConfigSFTPRemotePath); ok {
		keyValues["remote-path"] = configSFTPRemote.GetRemotePath()
	}

	return keyValues
}

// ConfigKeyNames returns sorted key names available for a config.
func ConfigKeyNames(config Config) []string {
	keyValues := ConfigKeyValues(config)
	keys := make([]string, 0, len(keyValues))

	for key := range keyValues {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

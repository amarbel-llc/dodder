package blob_stores

import (
	"path"
	"strings"

	"github.com/pkg/sftp"

	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/compression_type"
)

type DiscoveredConfig struct {
	HashTypeId string
	MultiHash  bool
	Buckets    []int
}

func DiscoverRemoteConfig(
	sftpClient *sftp.Client,
	remotePath string,
	uiPrinter ui.Printer,
) (discovered DiscoveredConfig, err error) {
	uiPrinter.Printf("discovering remote blob store config at %q...", remotePath)

	entries, err := sftpClient.ReadDir(remotePath)
	if err != nil {
		err = errors.Wrapf(err, "failed to read remote directory %q", remotePath)
		return discovered, err
	}

	// Check for multi-hash: look for subdirectories named after known hash types
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		if name == markl.FormatIdHashSha256 || name == markl.FormatIdHashBlake2b256 {
			discovered.MultiHash = true
			discovered.HashTypeId = name

			uiPrinter.Printf("detected multi-hash store with hash type %q", name)

			// Discover bucket depth from within the hash type directory
			hashDir := path.Join(remotePath, name)

			if discovered.Buckets, err = discoverBucketDepth(
				sftpClient,
				hashDir,
				uiPrinter,
			); err != nil {
				return discovered, err
			}

			return discovered, err
		}
	}

	// Single-hash: infer bucket depth from the directory structure directly
	discovered.MultiHash = false
	discovered.HashTypeId = blob_store_configs.DefaultHashTypeId

	if discovered.Buckets, err = discoverBucketDepth(
		sftpClient,
		remotePath,
		uiPrinter,
	); err != nil {
		return discovered, err
	}

	uiPrinter.Printf(
		"discovered config: hash=%s buckets=%v multi-hash=%t",
		discovered.HashTypeId,
		discovered.Buckets,
		discovered.MultiHash,
	)

	return discovered, err
}

func discoverBucketDepth(
	sftpClient *sftp.Client,
	dirPath string,
	uiPrinter ui.Printer,
) (buckets []int, err error) {
	// Walk down the directory tree counting directory levels with short names
	// (bucket directories) until we reach a file.
	currentPath := dirPath

	for {
		entries, readErr := sftpClient.ReadDir(currentPath)
		if readErr != nil {
			err = errors.Wrapf(readErr, "failed to read directory %q", currentPath)
			return buckets, err
		}

		if len(entries) == 0 {
			if len(buckets) == 0 {
				err = errors.Errorf(
					"remote directory %q is empty; cannot discover bucket structure",
					dirPath,
				)
			}

			return buckets, err
		}

		// Skip non-blob entries (like the config file)
		var firstDir string
		foundFile := false

		for _, entry := range entries {
			name := entry.Name()

			if name == directory_layout.FileNameBlobStoreConfig {
				continue
			}

			if strings.HasPrefix(name, "tmp_") {
				continue
			}

			if entry.IsDir() {
				if firstDir == "" {
					firstDir = name
				}

				continue
			}

			foundFile = true

			break
		}

		if foundFile {
			// Reached the file level, done
			return buckets, err
		}

		if firstDir == "" {
			err = errors.Errorf(
				"remote directory %q contains no blob files or bucket directories",
				currentPath,
			)
			return buckets, err
		}

		// This directory level is a bucket directory
		buckets = append(buckets, len(firstDir))
		currentPath = path.Join(currentPath, firstDir)
	}
}

func WriteRemoteConfig(
	sftpClient *sftp.Client,
	remotePath string,
	discovered DiscoveredConfig,
	uiPrinter ui.Printer,
) (err error) {
	configPath := path.Join(remotePath, directory_layout.FileNameBlobStoreConfig)

	uiPrinter.Printf("writing remote blob store config to %q...", configPath)

	config := &blob_store_configs.DefaultType{
		HashTypeId:        blob_store_configs.HashType(discovered.HashTypeId),
		HashBuckets:       discovered.Buckets,
		CompressionType:   compression_type.CompressionTypeDefault,
		LockInternalFiles: true,
	}

	typedConfig := &hyphence.TypedBlob[blob_store_configs.Config]{
		Type: ids.GetOrPanic(ids.TypeTomlBlobStoreConfigVCurrent).TypeStruct,
		Blob: config,
	}

	var configFile *sftp.File

	if configFile, err = sftpClient.Create(configPath); err != nil {
		err = errors.Wrapf(err, "failed to create remote config file %q", configPath)
		return err
	}

	defer errors.DeferredCloser(&err, configFile)

	if _, err = blob_store_configs.Coder.EncodeTo(typedConfig, configFile); err != nil {
		err = errors.Wrapf(err, "failed to write remote config to %q", configPath)
		return err
	}

	uiPrinter.Printf("remote blob store config written successfully")

	return err
}

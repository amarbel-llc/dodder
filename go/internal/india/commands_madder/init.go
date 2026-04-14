package commands_madder

import (
	"fmt"
	"io"
	"os"

	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/compression_type"
	tap "github.com/amarbel-llc/bob/packages/tap-dancer/go"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func init() {
	utility.AddCmd("init", &Init{
		tipe: ids.GetOrPanic(ids.TypeTomlBlobStoreConfigVCurrent).TypeStruct,
		blobStoreConfig: &blob_store_configs.DefaultType{
			CompressionType:   compression_type.CompressionTypeDefault,
			LockInternalFiles: true,
		},
		desc: command.Description{
			Short: "initialize a local blob store",
			Long: "Create a new local content-addressable blob store with " +
				"hash-bucketed directory layout. The store is registered " +
				"under the given store ID and uses the default compression " +
				"and hash settings. Requires a store ID argument " +
				"(e.g. 'default', '.archive').",
		},
	})

	utility.AddCmd("init-pointer", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigPointerV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlPointerV0{},
		desc: command.Description{
			Short: "initialize a pointer blob store",
			Long: "Create a blob store that delegates to another store by " +
				"reference. The pointer store does not hold blobs itself " +
				"but redirects reads and writes to the target store.",
		},
	})

	utility.AddCmd("init-sftp-explicit", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigSftpExplicitV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlSFTPV0{},
		desc: command.Description{
			Short: "initialize an SFTP blob store with explicit credentials",
			Long: "Create a blob store backed by an SFTP remote, using " +
				"explicitly provided host, port, user, and key path. " +
				"Use -discover to detect an existing remote store's " +
				"configuration from its directory structure.",
		},
	})

	utility.AddCmd("init-sftp-ssh_config", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlSFTPViaSSHConfigV0{},
		desc: command.Description{
			Short: "initialize an SFTP blob store via ssh_config",
			Long: "Create a blob store backed by an SFTP remote, resolving " +
				"connection parameters from ~/.ssh/config host entries. " +
				"Use -discover to detect an existing remote store's " +
				"configuration from its directory structure.",
		},
	})

	utility.AddCmd("init-inventory-archive", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveVCurrent,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV2{
			Delta: blob_store_configs.DeltaConfig{
				Enabled:     false,
				Algorithm:   "bsdiff",
				MinBlobSize: 256,
				MaxBlobSize: 10485760,
				SizeRatio:   2.0,
			},
		},
		desc: command.Description{
			Short: "initialize an inventory archive blob store",
			Long: "Create a blob store using the inventory archive format, " +
				"which packs blobs into indexed archive files for efficient " +
				"storage and O(1) lookups. This is the current archive " +
				"format version with optional delta compression support.",
		},
	})

	utility.AddCmd("init-inventory-archive-v1", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV1{
			Delta: blob_store_configs.DeltaConfig{
				Enabled:     false,
				Algorithm:   "bsdiff",
				MinBlobSize: 256,
				MaxBlobSize: 10485760,
				SizeRatio:   2.0,
			},
		},
		desc: command.Description{
			Short: "initialize an inventory archive blob store (v1)",
			Long: "Create a blob store using inventory archive format " +
				"version 1 with delta compression support. Prefer " +
				"init-inventory-archive for the current version.",
		},
	})

	utility.AddCmd("init-inventory-archive-v0", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV0{},
		desc: command.Description{
			Short: "initialize an inventory archive blob store (v0)",
			Long: "Create a blob store using the original inventory " +
				"archive format (v0) without delta compression. Prefer " +
				"init-inventory-archive for the current version.",
		},
	})
}

type Init struct {
	tipe            ids.TypeStruct
	blobStoreConfig blob_store_configs.ConfigMutable
	discover        bool
	desc            command.Description

	command_components_madder.EnvBlobStore
	command_components_madder.Init
}

var (
	_ interfaces.CommandComponentWriter = (*Init)(nil)
	_ command.CommandWithArgs           = (*Init)(nil)
)

func (cmd *Init) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "store-id",
			Description: "identifier for the new blob store (e.g. 'default', '.archive')",
			Required:    true,
		}},
	}}
}

func (cmd Init) GetDescription() command.Description {
	return cmd.desc
}

func (cmd *Init) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.blobStoreConfig.SetFlagDefinitions(flagDefinitions)

	if _, isSftp := cmd.blobStoreConfig.(blob_store_configs.ConfigSFTPRemotePath); isSftp {
		flagDefinitions.BoolVar(
			&cmd.discover,
			"discover",
			false,
			"Discover remote blob store config from existing directory structure",
		)
	}
}

func (cmd *Init) Run(req command.Request) {
	var blobStoreId blob_store_id.Id

	if err := blobStoreId.Set(req.PopArg("blob store id")); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
	}

	req.AssertNoMoreArgs()

	tw := tap.NewWriter(os.Stdout)

	if cmd.discover {
		cmd.runDiscover(req, blobStoreId, tw)
		return
	}

	envBlobStore := cmd.MakeEnvBlobStore(req)

	pathConfig := cmd.InitBlobStore(
		req,
		envBlobStore,
		blobStoreId,
		&blob_store_configs.TypedConfig{
			Type: cmd.tipe,
			Blob: cmd.blobStoreConfig,
		},
	)

	tw.Ok(fmt.Sprintf("init %s", pathConfig.GetConfig()))
	tw.Plan()
}

func (cmd *Init) runDiscover(
	req command.Request,
	blobStoreId blob_store_id.Id,
	tw *tap.Writer,
) {
	sftpConfig, ok := cmd.blobStoreConfig.(blob_store_configs.ConfigSFTPRemotePath)
	if !ok {
		errors.ContextCancelWithBadRequestError(
			req,
			errors.Errorf("--discover is only supported for SFTP blob stores"),
		)
		return
	}

	printer := ui.MakePrefixPrinter(
		ui.Err(),
		fmt.Sprintf("(blob_store: %s) ", blobStoreId),
	)

	// Connect to remote via SSH/SFTP
	var sshClient *ssh.Client
	var err error

	switch config := cmd.blobStoreConfig.(type) {
	case blob_store_configs.ConfigSFTPUri:
		if sshClient, err = blob_stores.MakeSSHClientFromSSHConfig(
			req,
			printer,
			config,
		); err != nil {
			errors.ContextCancelWithBadRequestError(req, err)
			return
		}

	case blob_store_configs.ConfigSFTPConfigExplicit:
		if sshClient, err = blob_stores.MakeSSHClientForExplicitConfig(
			req,
			printer,
			config,
		); err != nil {
			errors.ContextCancelWithBadRequestError(req, err)
			return
		}

	default:
		errors.ContextCancelWithBadRequestError(
			req,
			errors.Errorf("unsupported SFTP config type %T for --discover", config),
		)
		return
	}

	defer sshClient.Close() //defer:err-checked

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		errors.ContextCancelWithBadRequestError(
			req,
			errors.Wrapf(err, "failed to create SFTP client"),
		)
		return
	}

	defer sftpClient.Close() //defer:err-checked

	remotePath := sftpConfig.GetRemotePath()

	// Discover remote config from directory structure
	discovered, err := blob_stores.DiscoverRemoteConfig(sftpClient, remotePath, printer)
	if err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
		return
	}

	tw.Ok(fmt.Sprintf(
		"discovered config: hash=%s buckets=%v multi-hash=%t",
		discovered.HashTypeId,
		discovered.Buckets,
		discovered.MultiHash,
	))

	// Write config to remote
	if err = blob_stores.WriteRemoteConfig(
		sftpClient,
		remotePath,
		discovered,
		printer,
	); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
		return
	}

	tw.Ok("remote config written")

	// Write local SFTP config pointing to the remote
	envBlobStore := cmd.MakeEnvBlobStore(req)

	pathConfig := cmd.InitBlobStore(
		req,
		envBlobStore,
		blobStoreId,
		&blob_store_configs.TypedConfig{
			Type: cmd.tipe,
			Blob: cmd.blobStoreConfig,
		},
	)

	tw.Ok(fmt.Sprintf("init %s", pathConfig.GetConfig()))

	// Validate by reading a sample of blobs via the newly configured store
	configNamed := blob_store_configs.ConfigNamed{
		Path: pathConfig,
		Config: blob_store_configs.TypedConfig{
			Type: cmd.tipe,
			Blob: cmd.blobStoreConfig,
		},
	}

	blobStore := blob_stores.MakeRemoteBlobStore(envBlobStore, configNamed)

	var verifiedCount int

	for digest, iterErr := range blobStore.AllBlobs() {
		if iterErr != nil {
			tw.NotOk("blob iteration", map[string]string{"message": iterErr.Error()})
			break
		}

		if err = blob_stores.VerifyBlob(
			req,
			blobStore,
			digest,
			io.Discard,
		); err != nil {
			tw.NotOk(fmt.Sprintf("%s", digest), map[string]string{"message": err.Error()})
			break
		}

		verifiedCount++
		tw.Ok(fmt.Sprintf("verified %s", digest))

		if verifiedCount >= 5 {
			break
		}
	}

	tw.Comment(fmt.Sprintf("verified %d blobs", verifiedCount))
	tw.Plan()
}

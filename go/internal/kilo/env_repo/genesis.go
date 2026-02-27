package env_repo

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/india/genesis_configs"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
	"code.linenisgreat.com/dodder/go/lib/echo/ohio_files"
)

func (env *Env) Genesis(bigBang BigBang) {
	if env.directoryLayoutBlobStore == nil {
		errors.ContextCancelWithErrorf(
			env,
			"blob store directory layout not initialized",
		)
	}

	if env.Repo == nil {
		errors.ContextCancelWithErrorf(
			env,
			"repo directory layout not initialized",
		)
	}

	{
		privateKeyMutable := bigBang.GenesisConfig.Blob.GetPrivateKeyMutable()

		if bigBang.PrivateKey.IsNull() {
			if err := privateKeyMutable.GeneratePrivateKey(
				nil,
				markl.FormatIdEd25519Sec,
				markl.PurposeRepoPrivateKeyV1,
			); err != nil {
				env.Cancel(err)
				return
			}
		} else {
			if err := privateKeyMutable.SetPurposeId(
				markl.PurposeRepoPrivateKeyV1,
			); err != nil {
				env.Cancel(err)
				return
			}

			if err := privateKeyMutable.SetMarklId(
				bigBang.PrivateKey.GetMarklFormat().GetMarklFormatId(),
				bigBang.PrivateKey.GetBytes(),
			); err != nil {
				env.Cancel(err)
				return
			}
		}
	}

	bigBang.GenesisConfig.Blob.SetInventoryListTypeId(
		bigBang.InventoryListType.String(),
	)

	env.config.Type = bigBang.GenesisConfig.Type
	env.config.Blob = bigBang.GenesisConfig.Blob

	if err := env.MakeDirs(env.DirsGenesis()...); err != nil {
		env.Cancel(err)
		return
	}

	env.writeInventoryListLog()
	env.writeConfig(bigBang)
	env.writeBlobStoreConfigIfNecessary(bigBang, env.directoryLayoutBlobStore)

	if err := ohio_files.CopyFileLines(
		bigBang.Yin,
		filepath.Join(env.DirObjectId(), "Yin"),
	); err != nil {
		env.Cancel(err)
		return
	}

	if err := ohio_files.CopyFileLines(
		bigBang.Yang,
		filepath.Join(env.DirObjectId(), "Yang"),
	); err != nil {
		env.Cancel(err)
		return
	}

	env.writeFile(env.FileConfig(), "")
	env.writeFile(env.FileCacheDormant(), "")

	env.BlobStoreEnv = MakeBlobStoreEnv(
		env.Env,
	)
}

func (env Env) writeInventoryListLog() {
	var file *os.File

	{
		var err error

		if file, err = files.CreateExclusiveWriteOnly(
			env.FileInventoryListLog(),
		); err != nil {
			env.Cancel(err)
			return
		}

		defer errors.ContextMustClose(env, file)
	}

	coder := triple_hyphen_io.Coder[*triple_hyphen_io.TypedBlobEmpty]{
		Metadata: triple_hyphen_io.TypedMetadataCoder[struct{}]{},
	}

	tipe := ids.GetOrPanic(
		env.config.Blob.GetInventoryListTypeId(),
	).TypeStruct

	subject := triple_hyphen_io.TypedBlobEmpty{
		Type: tipe,
	}

	if _, err := coder.EncodeTo(&subject, file); err != nil {
		env.Cancel(err)
	}
}

func (env *Env) writeConfig(bigBang BigBang) {
	if err := triple_hyphen_io.EncodeToFile(
		genesis_configs.CoderPrivate,
		&env.config,
		env.GetPathConfigSeed().String(),
	); err != nil {
		env.Cancel(err)
		return
	}
}

func (env *Env) writeFile(path string, contents any) {
	var file *os.File

	{
		var err error

		if file, err = files.CreateExclusiveWriteOnly(path); err != nil {
			if errors.IsExist(err) {
				ui.Err().Printf("%s already exists, not overwriting", path)
				err = nil
			} else {
				env.Cancel(err)
				return
			}
		}
	}

	defer errors.ContextMustClose(env, file)

	if value, ok := contents.(string); ok {
		if _, err := io.WriteString(file, value); err != nil {
			env.Cancel(err)
			return
		}
	} else {
		// TODO remove gob
		enc := gob.NewEncoder(file)

		if err := enc.Encode(contents); err != nil {
			env.Cancel(err)
			return
		}
	}
}

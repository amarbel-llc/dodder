package env_repo

import (
	"io"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/delta/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
	mad_blob_store_env "github.com/amarbel-llc/madder/go/pkgs/blob_store_env"
	mad_directory_layout "github.com/amarbel-llc/madder/go/pkgs/directory_layout"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

func (env *Env) Genesis(bigBang BigBang) {
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
	env.writeBlobStoreConfigIfNecessary(bigBang, env.blobStoreEnv.BlobStore)
	env.writeBlobStoreConfigInit(bigBang, env.blobStoreEnv.BlobStore)

	// Re-make the blob_store_env now that the on-disk config exists,
	// so store discovery picks up the freshly-written config. Reuses
	// the env_local embedded in the existing blobStoreEnv to avoid
	// re-running env_dir.initializeXDG.
	env.blobStoreEnv = mad_blob_store_env.MakeBlobStoreEnv(env.blobStoreEnv.Env)

	env.genesisObjectIds(bigBang)

	env.writeFile(env.FileConfig(), "")
	env.writeFile(env.FileCacheDormant(), "")
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

	coder := hyphence.Coder[*hyphence.TypedBlobEmpty]{
		Metadata: hyphence.TypedMetadataCoder[struct{}]{},
	}

	tipe := ids.GetOrPanic(
		env.config.Blob.GetInventoryListTypeId(),
	).TypeStruct

	subject := hyphence.TypedBlobEmpty{
		Type: tipe.ToMadder(),
	}

	if _, err := coder.EncodeTo(&subject, file); err != nil {
		env.Cancel(err)
	}
}

func (env *Env) writeConfig(bigBang BigBang) {
	if err := hyphence.EncodeToFile(
		genesis_configs.CoderPrivate,
		&env.config,
		env.GetPathConfigSeed().String(),
	); err != nil {
		env.Cancel(err)
		return
	}
}

// writeBlobStoreConfigIfNecessary writes the initial blob store
// config to disk when the genesis bigBang did not specify a
// pre-existing blob store id. Moved here from the deleted
// env_repo/blob_store.go: it operates on the env_repo (uses
// env.MakeDirs / env.Cancel) and on madder's directory_layout, so
// belonging to env_repo as a method is the natural home.
func (env *Env) writeBlobStoreConfigIfNecessary(
	bigBang BigBang,
	directoryLayout mad_directory_layout.BlobStore,
) {
	if !bigBang.BlobStoreId.IsEmpty() {
		return
	}

	blobStoreConfigPath := mad_directory_layout.GetDefaultBlobStore(
		directoryLayout,
	).GetConfig()

	blobStoreConfigDir := filepath.Dir(blobStoreConfigPath)

	if err := env.MakeDirs(blobStoreConfigDir); err != nil {
		env.Cancel(err)
		return
	}

	blobStoreConfig := bigBang.TypedBlobStoreConfig

	if err := hyphence.EncodeToFile(
		blob_store_configs.Coder,
		&blob_store_configs.TypedConfig{
			Type: blobStoreConfig.Type,
			Blob: blobStoreConfig.Blob,
		},
		blobStoreConfigPath,
	); err != nil {
		env.Cancel(err)
		return
	}
}

// writeBlobStoreConfigInit writes bigBang.BlobStoreConfigInit to disk at
// bigBang.BlobStoreId's blob_store-config path. Used by init-workspace
// to install a pointer-store config under .<workspace-name>/ before
// blob-store discovery re-runs. Skipped when either field is unset.
func (env *Env) writeBlobStoreConfigInit(
	bigBang BigBang,
	directoryLayout mad_directory_layout.BlobStore,
) {
	if bigBang.BlobStoreConfigInit == nil {
		return
	}

	if bigBang.BlobStoreId.IsEmpty() {
		return
	}

	blobStorePath := mad_directory_layout.GetBlobStorePath(
		directoryLayout,
		bigBang.BlobStoreId.String(),
	)
	blobStoreConfigPath := blobStorePath.GetConfig()
	blobStoreConfigDir := filepath.Dir(blobStoreConfigPath)

	if err := env.MakeDirs(blobStoreConfigDir); err != nil {
		env.Cancel(err)
		return
	}

	if err := hyphence.EncodeToFile(
		blob_store_configs.Coder,
		&blob_store_configs.TypedConfig{
			Type: bigBang.BlobStoreConfigInit.Type,
			Blob: bigBang.BlobStoreConfigInit.Blob,
		},
		blobStoreConfigPath,
	); err != nil {
		env.Cancel(err)
		return
	}
}

func (env *Env) writeFile(path string, contents string) {
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

	if _, err := io.WriteString(file, contents); err != nil {
		env.Cancel(err)
		return
	}
}

func (env *Env) genesisObjectIds(bigBang BigBang) {
	if bigBang.Yin == "" && bigBang.Yang == "" {
		return
	}

	yinSlice := readAndCleanFileLines(env, bigBang.Yin)
	yangSlice := enforceCrossSideUniqueness(yinSlice, readAndCleanFileLines(env, bigBang.Yang))

	yinBlobId := genesisWriteWordsAsBlob(env, yinSlice)
	yangBlobId := genesisWriteWordsAsBlob(env, yangSlice)

	tai := ids.NowTai()
	log := zettel_id_log.Log{Path: env.FileZettelIdLog()}

	yinEntry := &zettel_id_log.V1{
		Side:      zettel_id_log.SideYin,
		Tai:       tai,
		MarklId:   yinBlobId,
		WordCount: len(yinSlice),
	}

	if err := log.AppendEntry(yinEntry); err != nil {
		env.Cancel(err)
		return
	}

	yangEntry := &zettel_id_log.V1{
		Side:      zettel_id_log.SideYang,
		Tai:       tai,
		MarklId:   yangBlobId,
		WordCount: len(yangSlice),
	}

	if err := log.AppendEntry(yangEntry); err != nil {
		env.Cancel(err)
		return
	}

	genesisWriteFlatFile(env, filepath.Join(env.DirObjectId(), zettel_id_provider.FilePathZettelIdYin), yinSlice)
	genesisWriteFlatFile(env, filepath.Join(env.DirObjectId(), zettel_id_provider.FilePathZettelIdYang), yangSlice)
}

func readAndCleanFileLines(env *Env, filePath string) []string {
	file, err := files.Open(filePath)
	if err != nil {
		env.Cancel(err)
		return nil
	}

	defer errors.ContextMustClose(env, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	seen := make(map[string]struct{})
	var words []string

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			env.Cancel(errIter)
			return nil
		}

		cleaned := zettel_id_provider.Clean(line)

		if cleaned == "" {
			continue
		}

		if _, ok := seen[cleaned]; ok {
			continue
		}

		seen[cleaned] = struct{}{}
		words = append(words, cleaned)
	}

	return words
}

func enforceCrossSideUniqueness(yin, yang []string) []string {
	yinSet := make(map[string]struct{}, len(yin))
	for _, word := range yin {
		yinSet[word] = struct{}{}
	}

	result := make([]string, 0, len(yang))

	for _, word := range yang {
		if _, ok := yinSet[word]; !ok {
			result = append(result, word)
		}
	}

	return result
}

func genesisWriteWordsAsBlob(env *Env, words []string) markl.Id {
	blobWriter, err := env.GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		env.Cancel(err)
		return markl.Id{}
	}

	defer errors.ContextMustClose(env, blobWriter)

	for _, word := range words {
		if _, err := io.WriteString(blobWriter, word); err != nil {
			env.Cancel(err)
			return markl.Id{}
		}

		if _, err := io.WriteString(blobWriter, "\n"); err != nil {
			env.Cancel(err)
			return markl.Id{}
		}
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id
}

func genesisWriteFlatFile(env *Env, filePath string, words []string) {
	file, err := files.CreateExclusiveWriteOnly(filePath)
	if err != nil {
		env.Cancel(err)
		return
	}

	defer errors.ContextMustClose(env, file)

	for _, word := range words {
		if _, err := io.WriteString(file, word); err != nil {
			env.Cancel(err)
			return
		}

		if _, err := io.WriteString(file, "\n"); err != nil {
			env.Cancel(err)
			return
		}
	}
}

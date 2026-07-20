package env_repo

import (
	"io"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/delta/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/echo/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	mad_blob_store_env "code.linenisgreat.com/madder/go/pkgs/blob_store_env"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_id"
	mad_directory_layout "code.linenisgreat.com/madder/go/pkgs/directory_layout"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
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

	// Pre-flight: if the caller supplied -blob_store-id, the store must
	// already exist on disk. writeBlobStoreConfigIfNecessary trusts the
	// caller and early-returns without creating one, and subsequent
	// commands will panic in env_repo.GetReadBlobStore with "no write
	// store given" if the named store can't be resolved (#214).
	//
	// Make() calls MakeBlobStoreEnvWithoutStores for a fresh repo
	// (no config on disk yet), so env.blobStoreEnv currently has zero
	// stores enumerated. Force a real rebuild so the lookup can see any
	// pre-existing user/system stores (e.g. one created via `madder
	// init shared` before this dodder init), then validate that the
	// named id resolves. Subsequent code paths perform another rebuild
	// after writing the default-store config, so this early rebuild is
	// purely for the lookup-and-validate step.
	//
	// Skip when BlobStoreConfigInit is non-nil: the caller is creating
	// the store as part of this Genesis (e.g. init-workspace writing a
	// TomlPointerV1 to its parent), so requiring the store to pre-exist
	// would be a chicken-and-egg failure.
	if !bigBang.BlobStoreId.IsEmpty() && bigBang.BlobStoreConfigInit == nil {
		env.blobStoreEnv = mad_blob_store_env.MakeBlobStoreEnv(env.blobStoreEnv.Env)

		// BlobStoreInitialized is a struct with two embedded interface
		// fields; a missing store comes back with both nil. Probing
		// the BlobStore field is sufficient since every real store
		// has one.
		store := env.blobStoreEnv.GetBlobStore(bigBang.BlobStoreId)
		if store.BlobStore == nil {
			env.Cancel(errors.ErrorWithStackf(
				"blob store %q not found; create it first (e.g. `madder init %q` or `dodder blob_store-init %q`) before running init with -blob_store-id",
				bigBang.BlobStoreId,
				bigBang.BlobStoreId,
				bigBang.BlobStoreId,
			))
			return
		}
	}

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

	// Pin the caller-supplied store as the default for every write the
	// rest of Genesis (and callers building on top of it, e.g.
	// local_working_copy.Genesis's pandoc tool blobs) makes from here
	// on. Without this, MakeBlobStoreEnv's own default-selection
	// (madder's BlobStoreEnv.setupStores, an alphabetical sort of every
	// store discovered in the XDG scope) silently wins instead, so
	// writes land in whatever store happens to sort first rather than
	// the one -blob_store-id named (amarbel-llc/dodder#365).
	if !bigBang.BlobStoreId.IsEmpty() {
		env.SetBlobStoreOrder([]blob_store_id.Id{bigBang.BlobStoreId})
	}

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

	// FDR-0019: named repos share the default (madder) blob store, so
	// the config may already exist when a second named repo is
	// initialized in the same scope. Reuse it rather than failing the
	// exclusive write below. Legacy single-repo inits never hit this
	// path — their blob store tree is unique per repo.
	if files.Exists(blobStoreConfigPath) {
		return
	}

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

// writeBlobStoreConfigInit writes the caller-supplied
// BlobStoreConfigInit (e.g. a TomlPointerV1 set by init-workspace
// per #200) to disk at the blob store id's config path. Skipped when
// either field is unset. Pairs with the pre-flight skip in Genesis:
// the caller passes BlobStoreConfigInit + BlobStoreId together to
// install a not-yet-existing store as part of the init.
func (env *Env) writeBlobStoreConfigInit(
	bigBang BigBang,
	directoryLayout mad_directory_layout.BlobStore,
) {
	if bigBang.BlobStoreConfigInit == nil || bigBang.BlobStoreId.IsEmpty() {
		return
	}

	// IMPORTANT: pass the BARE name (e.g. "workspace-repo-id"),
	// NOT the canonical String form (".workspace-repo-id").
	// `MakeWithLocation` in madder's discovery layer does not strip
	// a leading dot from the raw value, so writing to a path
	// `.workspace-repo-id/` would produce a discovery key
	// "..workspace-repo-id" (double dot) and never match the konfig's
	// `.workspace-repo-id` reference. The default store follows the
	// same rule: written to `default/`, discovered as Id "default"
	// (Cwd-scoped, single-dot String). See blob_store_id.MakeWithLocation
	// and Id.String() (madder/internal/alfa/blob_store_id/main.go).
	blobStorePath := mad_directory_layout.GetBlobStorePath(
		directoryLayout,
		bigBang.BlobStoreId.GetName(),
	)
	blobStoreConfigPath := blobStorePath.GetConfig()
	blobStoreConfigDir := filepath.Dir(blobStoreConfigPath)

	if err := env.MakeDirs(blobStoreConfigDir); err != nil {
		env.Cancel(err)
		return
	}

	blobStoreConfig := bigBang.BlobStoreConfigInit

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
	yinHasSource := bigBang.Yin != "" || bigBang.YinDefault
	yangHasSource := bigBang.Yang != "" || bigBang.YangDefault

	if !yinHasSource && !yangHasSource {
		return
	}

	yinSlice := env.genesisLoadSideWords(
		bigBang.Yin,
		bigBang.YinDefault,
		zettel_id_provider.DefaultYinReader(),
	)
	yangSlice := enforceCrossSideUniqueness(
		yinSlice,
		env.genesisLoadSideWords(
			bigBang.Yang,
			bigBang.YangDefault,
			zettel_id_provider.DefaultYangReader(),
		),
	)

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

// genesisLoadSideWords loads one side's zettel-id words: from filePath
// when set, else from the embedded default (defaultReader) when
// useDefault, else nil (the side has no source).
func (env *Env) genesisLoadSideWords(
	filePath string,
	useDefault bool,
	defaultReader io.Reader,
) []string {
	if filePath != "" {
		return readAndCleanFileLines(env, filePath)
	}

	if useDefault {
		return readAndCleanReader(env, defaultReader)
	}

	return nil
}

func readAndCleanFileLines(env *Env, filePath string) []string {
	file, err := files.Open(filePath)
	if err != nil {
		env.Cancel(err)
		return nil
	}

	defer errors.ContextMustClose(env, file)

	return readAndCleanReader(env, file)
}

func readAndCleanReader(env *Env, reader io.Reader) []string {
	bufferedReader, repool := pool.GetBufferedReader(reader)
	defer repool()

	seen := make(map[string]struct{})
	var words []string

	for line, errIter := range ohio.MakeLineSeqFromReader(bufferedReader) {
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

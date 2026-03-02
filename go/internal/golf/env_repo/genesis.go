package env_repo

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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

	env.BlobStoreEnv = MakeBlobStoreEnv(
		env.Env,
	)

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

func (env *Env) genesisObjectIds(bigBang BigBang) {
	if bigBang.Yin == "" && bigBang.Yang == "" {
		return
	}

	yinWords := readAndCleanFileLines(env, bigBang.Yin)
	yangWords := readAndCleanFileLines(env, bigBang.Yang)

	enforceCrossSideUniqueness(yinWords, yangWords)

	yinSlice := orderedKeys(yinWords)
	yangSlice := orderedKeys(yangWords)

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

func readAndCleanFileLines(env *Env, filePath string) map[string]struct{} {
	file, err := files.Open(filePath)
	if err != nil {
		env.Cancel(err)
		return nil
	}

	defer errors.ContextMustClose(env, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	words := make(map[string]struct{})

	for line, errIter := range ohio.MakeLineSeqFromReader(reader) {
		if errIter != nil {
			env.Cancel(errIter)
			return nil
		}

		cleaned := zettel_id_provider.Clean(line)

		if cleaned == "" {
			continue
		}

		words[cleaned] = struct{}{}
	}

	return words
}

func enforceCrossSideUniqueness(yin, yang map[string]struct{}) {
	for word := range yin {
		delete(yang, word)
	}
}

func orderedKeys(m map[string]struct{}) []string {
	result := make([]string, 0, len(m))

	for k := range m {
		result = append(result, k)
	}

	sort.Strings(result)

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

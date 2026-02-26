package env_repo

import (
	"encoding/gob"
	"io"
	"os"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	pool "code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_log"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_provider"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/src/hotel/genesis_configs"
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

		if err := privateKeyMutable.GeneratePrivateKey(
			nil,
			markl.FormatIdEd25519Sec,
			markl.PurposeRepoPrivateKeyV1,
		); err != nil {
			env.Cancel(err)
			return
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

	yinWords, err := env.readAndCleanFileLines(bigBang.Yin)
	if err != nil {
		env.Cancel(err)
		return
	}

	yangWords, err := env.readAndCleanFileLines(bigBang.Yang)
	if err != nil {
		env.Cancel(err)
		return
	}

	env.enforceCrossSideUniqueness(yinWords, yangWords)

	yinBlobId, err := env.genesisWriteWordsAsBlob(yinWords)
	if err != nil {
		env.Cancel(err)
		return
	}

	yangBlobId, err := env.genesisWriteWordsAsBlob(yangWords)
	if err != nil {
		env.Cancel(err)
		return
	}

	log := object_id_log.Log{Path: env.FileObjectIdLog()}

	yinEntry := &object_id_log.V1{
		Side:      object_id_log.SideYin,
		Tai:       ids.NowTai(),
		MarklId:   yinBlobId,
		WordCount: len(yinWords),
	}

	if err := log.AppendEntry(yinEntry); err != nil {
		env.Cancel(err)
		return
	}

	yangEntry := &object_id_log.V1{
		Side:      object_id_log.SideYang,
		Tai:       ids.NowTai(),
		MarklId:   yangBlobId,
		WordCount: len(yangWords),
	}

	if err := log.AppendEntry(yangEntry); err != nil {
		env.Cancel(err)
		return
	}

	yinFlatPath := filepath.Join(env.DirObjectId(), object_id_provider.FilePathZettelIdYin)
	yangFlatPath := filepath.Join(env.DirObjectId(), object_id_provider.FilePathZettelIdYang)

	if err := env.genesisWriteFlatFile(yinFlatPath, yinWords); err != nil {
		env.Cancel(err)
		return
	}

	if err := env.genesisWriteFlatFile(yangFlatPath, yangWords); err != nil {
		env.Cancel(err)
		return
	}
}

func (env *Env) readAndCleanFileLines(path string) (words map[string]bool, err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return words, err
	}

	defer errors.DeferredCloser(&err, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	words = make(map[string]bool)

	for {
		var line string

		if line, err = reader.ReadString('\n'); len(line) > 0 {
			cleaned := object_id_provider.Clean(strings.TrimRight(line, "\n"))

			if cleaned != "" {
				words[cleaned] = true
			}
		}

		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}

			err = errors.Wrap(err)
			return words, err
		}
	}

	return words, err
}

func (env *Env) enforceCrossSideUniqueness(yin, yang map[string]bool) {
	for w := range yin {
		if yang[w] {
			delete(yin, w)
			delete(yang, w)
		}
	}
}

func (env *Env) genesisWriteWordsAsBlob(words map[string]bool) (id markl.Id, err error) {
	blobWriter, err := env.GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		err = errors.Wrap(err)
		return id, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	for word := range words {
		if _, err = io.WriteString(blobWriter, word); err != nil {
			err = errors.Wrap(err)
			return id, err
		}

		if _, err = io.WriteString(blobWriter, "\n"); err != nil {
			err = errors.Wrap(err)
			return id, err
		}
	}

	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id, err
}

func (env *Env) genesisWriteFlatFile(path string, words map[string]bool) (err error) {
	var file *os.File

	if file, err = files.CreateExclusiveWriteOnly(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	for word := range words {
		if _, err = io.WriteString(file, word); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if _, err = io.WriteString(file, "\n"); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

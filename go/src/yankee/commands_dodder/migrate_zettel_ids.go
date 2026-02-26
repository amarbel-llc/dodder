package commands_dodder

import (
	"io"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	pool "code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_log"
	"code.linenisgreat.com/dodder/go/src/foxtrot/object_id_provider"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("migrate-zettel-ids", &MigrateZettelIds{})
}

type MigrateZettelIds struct {
	command_components_dodder.LocalWorkingCopy
}

func (cmd MigrateZettelIds) Run(req command.Request) {
	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	envRepo := localWorkingCopy.GetEnvRepo()
	log := object_id_log.Log{Path: envRepo.FileObjectIdLog()}

	entries, err := log.ReadAllEntries()
	if err != nil {
		errors.ContextCancelWithErrorf(req, "reading object id log: %s", err)
		return
	}

	if len(entries) > 0 {
		ui.Out().Print("object id log already contains entries, skipping migration")
		return
	}

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	blobStore := envRepo.GetDefaultBlobStore()
	dirObjectId := envRepo.DirObjectId()
	tai := ids.NowTai()

	sides := []struct {
		side     object_id_log.Side
		fileName string
	}{
		{object_id_log.SideYin, object_id_provider.FilePathZettelIdYin},
		{object_id_log.SideYang, object_id_provider.FilePathZettelIdYang},
	}

	for _, s := range sides {
		flatPath := path.Join(dirObjectId, s.fileName)
		marklId, wordCount := writeFlatFileAsBlob(req, blobStore, flatPath)

		entry := &object_id_log.V1{
			Side:      s.side,
			Tai:       tai,
			MarklId:   marklId,
			WordCount: wordCount,
		}

		if err := log.AppendEntry(entry); err != nil {
			errors.ContextCancelWithErrorf(req, "appending %s log entry: %s", s.fileName, err)
			return
		}

		ui.Out().Printf("migrated %s: %d words, %s", s.fileName, wordCount, marklId)
	}
}

func writeFlatFileAsBlob(
	req command.Request,
	blobStore domain_interfaces.BlobStore,
	flatFilePath string,
) (markl.Id, int) {
	file, err := files.Open(flatFilePath)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	defer errors.ContextMustClose(req, file)

	reader, repool := pool.GetBufferedReader(file)
	defer repool()

	var wordCount int

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			if strings.TrimRight(line, "\n") != "" {
				wordCount++
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			errors.ContextCancelWithError(req, err)
			return markl.Id{}, 0
		}
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	blobWriter, err := blobStore.MakeBlobWriter(nil)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	defer errors.ContextMustClose(req, blobWriter)

	if _, err := io.Copy(blobWriter, file); err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}, 0
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id, wordCount
}

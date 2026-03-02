package commands_dodder

import (
	"io"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_log"
	"code.linenisgreat.com/dodder/go/internal/charlie/zettel_id_provider"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
	log := zettel_id_log.Log{Path: envRepo.FileZettelIdLog()}

	entries, err := log.ReadAllEntries()
	if err != nil {
		errors.ContextCancelWithErrorf(req, "reading zettel id log: %s", err)
		return
	}

	if len(entries) > 0 {
		ui.Out().Print("zettel id log already contains entries, skipping migration")
		return
	}

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	blobStore := envRepo.GetDefaultBlobStore()
	dirObjectId := envRepo.DirObjectId()
	tai := ids.NowTai()

	sides := []struct {
		side     zettel_id_log.Side
		fileName string
	}{
		{zettel_id_log.SideYin, zettel_id_provider.FilePathZettelIdYin},
		{zettel_id_log.SideYang, zettel_id_provider.FilePathZettelIdYang},
	}

	for _, s := range sides {
		flatPath := path.Join(dirObjectId, s.fileName)
		marklId, wordCount := writeFlatFileAsBlob(req, blobStore, flatPath)

		entry := &zettel_id_log.V1{
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

	for line, err := range ohio.MakeLineSeqFromReader(reader) {
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}, 0
		}

		if strings.TrimRight(line, "\n") != "" {
			wordCount++
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

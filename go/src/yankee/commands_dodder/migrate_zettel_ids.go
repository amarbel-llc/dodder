package commands_dodder

import (
	"bufio"
	"io"
	"os"
	"path"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
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
	logPath := envRepo.FileObjectIdLog()

	entries, err := object_id_log.ReadAllEntries(logPath)

	if err != nil {
		errors.ContextCancelWithErrorf(req, "reading object id log: %s", err)
		return
	}

	if len(entries) > 0 {
		ui.Out().Print("object id log already contains entries, skipping migration")
		return
	}

	blobStore := envRepo.GetDefaultBlobStore()
	dirObjectId := envRepo.DirObjectId()

	yinPath := path.Join(dirObjectId, object_id_provider.FilePathZettelIdYin)
	yangPath := path.Join(dirObjectId, object_id_provider.FilePathZettelIdYang)

	yinMarklId, yinWordCount, err := writeFlatFileAsBlob(blobStore, yinPath)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "writing yin blob: %s", err)
		return
	}

	yangMarklId, yangWordCount, err := writeFlatFileAsBlob(blobStore, yangPath)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "writing yang blob: %s", err)
		return
	}

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	tai := ids.NowTai()

	yinEntry := &object_id_log.V1{
		Side:      object_id_log.SideYin,
		Tai:       tai,
		MarklId:   yinMarklId,
		WordCount: yinWordCount,
	}

	if err := object_id_log.AppendEntry(logPath, yinEntry); err != nil {
		errors.ContextCancelWithErrorf(req, "appending yin log entry: %s", err)
		return
	}

	yangEntry := &object_id_log.V1{
		Side:      object_id_log.SideYang,
		Tai:       tai,
		MarklId:   yangMarklId,
		WordCount: yangWordCount,
	}

	if err := object_id_log.AppendEntry(logPath, yangEntry); err != nil {
		errors.ContextCancelWithErrorf(req, "appending yang log entry: %s", err)
		return
	}

	ui.Out().Printf(
		"migrated zettel ids: yin (%d words, %s), yang (%d words, %s)",
		yinWordCount,
		yinMarklId,
		yangWordCount,
		yangMarklId,
	)
}

func writeFlatFileAsBlob(
	blobStore domain_interfaces.BlobStore,
	flatFilePath string,
) (id markl.Id, wordCount int, err error) {
	var file *os.File

	if file, err = files.Open(flatFilePath); err != nil {
		err = errors.Wrap(err)
		return id, wordCount, err
	}

	defer errors.DeferredCloser(&err, file)

	wordCount, err = countLines(flatFilePath)
	if err != nil {
		err = errors.Wrap(err)
		return id, wordCount, err
	}

	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return id, wordCount, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = io.Copy(blobWriter, file); err != nil {
		err = errors.Wrap(err)
		return id, wordCount, err
	}

	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id, wordCount, err
}

func countLines(path string) (count int, err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return count, err
	}

	defer errors.DeferredCloser(&err, file)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			count++
		}
	}

	if err = scanner.Err(); err != nil {
		err = errors.Wrap(err)
		return count, err
	}

	return count, err
}

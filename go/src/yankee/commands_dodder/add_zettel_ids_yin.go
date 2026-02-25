package commands_dodder

import (
	"bufio"
	"io"
	"os"
	"path"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/unicorn"
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
	utility.AddCmd("add-zettel-ids-yin", &AddZettelIds{
		side:         object_id_log.SideYin,
		flatFileName: object_id_provider.FilePathZettelIdYin,
	})
}

type AddZettelIds struct {
	command_components_dodder.LocalWorkingCopy
	side         object_id_log.Side
	flatFileName string
}

func (cmd AddZettelIds) Run(req command.Request) {
	req.AssertNoMoreArgs()

	candidates := readAndExtractCandidates(req)

	localWorkingCopy := cmd.MakeLocalWorkingCopyWithOptions(
		req,
		env_ui.Options{},
		local_working_copy.OptionsAllowConfigReadError,
	)

	envRepo := localWorkingCopy.GetEnvRepo()
	dirObjectId := envRepo.DirObjectId()

	prov, err := object_id_provider.New(envRepo)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "loading zettel id provider: %s", err)
		return
	}

	existingWords := collectExistingWords(prov)

	var filtered []string

	for _, word := range candidates {
		cleaned := object_id_provider.Clean(word)

		if cleaned == "" {
			continue
		}

		if !existingWords[cleaned] {
			filtered = append(filtered, cleaned)
		}
	}

	if len(filtered) == 0 {
		ui.Out().Print("no new words to add")
		return
	}

	blobStore := envRepo.GetDefaultBlobStore()

	blobId, err := writeWordsAsBlob(blobStore, filtered)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "writing blob: %s", err)
		return
	}

	lockSmith := envRepo.GetLockSmith()

	if err := lockSmith.Lock(); err != nil {
		errors.ContextCancelWithErrorf(req, "acquiring lock: %s", err)
		return
	}

	defer func() {
		if err := lockSmith.Unlock(); err != nil {
			errors.ContextCancelWithErrorf(req, "releasing lock: %s", err)
		}
	}()

	logPath := envRepo.FileObjectIdLog()
	flatFilePath := path.Join(dirObjectId, cmd.flatFileName)

	entry := &object_id_log.V1{
		Side:      cmd.side,
		Tai:       ids.NowTai(),
		MarklId:   blobId,
		WordCount: len(filtered),
	}

	if err := object_id_log.AppendEntry(logPath, entry); err != nil {
		errors.ContextCancelWithErrorf(req, "appending log entry: %s", err)
		return
	}

	if err := appendWordsToFlatFile(flatFilePath, filtered); err != nil {
		errors.ContextCancelWithErrorf(req, "updating flat file cache: %s", err)
		return
	}

	yinCount := prov.Left().Len()
	yangCount := prov.Right().Len()

	if cmd.side == object_id_log.SideYin {
		yinCount += len(filtered)
	} else {
		yangCount += len(filtered)
	}

	poolSize := yinCount * yangCount

	ui.Out().Printf(
		"added %d words to %s (pool size: %d)",
		len(filtered),
		cmd.flatFileName,
		poolSize,
	)
}

func readAndExtractCandidates(req command.Request) []string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for {
		line, err := reader.ReadString('\n')

		if err != nil && err != io.EOF {
			errors.ContextCancelWithError(req, err)
		}

		if len(line) > 0 {
			line = strings.TrimRight(line, "\n")
			lines = append(lines, line)
		}

		if err == io.EOF {
			break
		}
	}

	return unicorn.ExtractUniqueComponents(lines)
}

func collectExistingWords(prov *object_id_provider.Provider) map[string]bool {
	existing := make(map[string]bool)

	for _, word := range prov.Left() {
		existing[word] = true
	}

	for _, word := range prov.Right() {
		existing[word] = true
	}

	return existing
}

func writeWordsAsBlob(
	blobStore domain_interfaces.BlobStore,
	words []string,
) (id markl.Id, err error) {
	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return id, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	content := strings.Join(words, "\n") + "\n"

	if _, err = io.WriteString(blobWriter, content); err != nil {
		err = errors.Wrap(err)
		return id, err
	}

	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id, err
}

func appendWordsToFlatFile(flatFilePath string, words []string) (err error) {
	var file *os.File

	if file, err = files.OpenFile(
		flatFilePath,
		os.O_WRONLY|os.O_APPEND,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	content := strings.Join(words, "\n") + "\n"

	if _, err = io.WriteString(file, content); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

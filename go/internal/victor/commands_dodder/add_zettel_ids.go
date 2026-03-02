package commands_dodder

import (
	"bufio"
	"io"
	"os"
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
	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd("add-zettel-ids-yin", &AddZettelIds{
		side:         zettel_id_log.SideYin,
		flatFileName: zettel_id_provider.FilePathZettelIdYin,
	})

	utility.AddCmd("add-zettel-ids-yang", &AddZettelIds{
		side:         zettel_id_log.SideYang,
		flatFileName: zettel_id_provider.FilePathZettelIdYang,
	})
}

type AddZettelIds struct {
	command_components_dodder.LocalWorkingCopy
	side         zettel_id_log.Side
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

	prov, err := zettel_id_provider.New(envRepo)
	if err != nil {
		errors.ContextCancelWithErrorf(req, "loading zettel id provider: %s", err)
		return
	}

	existingWords := collectExistingWords(prov)

	var filtered []string

	for _, word := range candidates {
		cleaned := zettel_id_provider.Clean(word)

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

	blobId := writeWordsAsBlob(req, envRepo.GetDefaultBlobStore(), filtered)

	lockSmith := envRepo.GetLockSmith()

	req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Lock))
	defer req.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	log := zettel_id_log.Log{Path: envRepo.FileZettelIdLog()}
	flatFilePath := path.Join(dirObjectId, cmd.flatFileName)

	entry := &zettel_id_log.V1{
		Side:      cmd.side,
		Tai:       ids.NowTai(),
		MarklId:   blobId,
		WordCount: len(filtered),
	}

	if err := log.AppendEntry(entry); err != nil {
		errors.ContextCancelWithErrorf(req, "appending log entry: %s", err)
		return
	}

	appendWordsToFlatFile(req, flatFilePath, filtered)

	yinCount := prov.Left().Len()
	yangCount := prov.Right().Len()

	if cmd.side == zettel_id_log.SideYin {
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

	for line, err := range ohio.MakeLineSeqFromReader(reader) {
		if err != nil {
			errors.ContextCancelWithError(req, err)
			return nil
		}

		lines = append(lines, strings.TrimRight(line, "\n"))
	}

	return unicorn.ExtractUniqueComponents(lines)
}

func collectExistingWords(prov *zettel_id_provider.Provider) map[string]bool {
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
	req command.Request,
	blobStore domain_interfaces.BlobStore,
	words []string,
) markl.Id {
	blobWriter, err := blobStore.MakeBlobWriter(nil)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return markl.Id{}
	}

	defer errors.ContextMustClose(req, blobWriter)

	for _, word := range words {
		if _, err := io.WriteString(blobWriter, word); err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}
		}

		if _, err := io.WriteString(blobWriter, "\n"); err != nil {
			errors.ContextCancelWithError(req, err)
			return markl.Id{}
		}
	}

	var id markl.Id
	id.ResetWithMarklId(blobWriter.GetMarklId())

	return id
}

func appendWordsToFlatFile(req command.Request, flatFilePath string, words []string) {
	file, err := files.OpenFile(
		flatFilePath,
		os.O_WRONLY|os.O_APPEND,
		0o666,
	)
	if err != nil {
		errors.ContextCancelWithError(req, err)
		return
	}

	defer errors.ContextMustClose(req, file)

	for _, word := range words {
		if _, err := io.WriteString(file, word); err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}

		if _, err := io.WriteString(file, "\n"); err != nil {
			errors.ContextCancelWithError(req, err)
			return
		}
	}
}

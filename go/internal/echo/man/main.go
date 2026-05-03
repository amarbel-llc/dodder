package man

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type PageConfig struct {
	BinaryName      string
	Section         int
	Version         string
	Source          string
	Description     string
	LongDescription string
	Date            string
}

func (config *PageConfig) resolveDate() {
	if config.Date != "" {
		return
	}

	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		if secs, err := strconv.ParseInt(epoch, 10, 64); err == nil {
			config.Date = time.Unix(secs, 0).UTC().Format("January 2006")
			return
		}
	}

	config.Date = time.Now().Format("January 2006")
}

func GenerateAll(
	config PageConfig,
	utility command.Utility,
	outputDir string,
) (err error) {
	config.resolveDate()

	// Generate main utility page
	mainPagePath := filepath.Join(
		outputDir,
		fmt.Sprintf("%s.%d", config.BinaryName, config.Section),
	)

	mainFile, err := os.Create(mainPagePath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", mainPagePath, err)
	}

	defer errors.DeferredCloser(&err, mainFile)

	if err := generateUtilityPage(mainFile, config, utility); err != nil {
		return fmt.Errorf("generating %s: %w", mainPagePath, err)
	}

	// Generate per-subcommand pages
	for name, cmd := range utility.AllCmds() {
		pagePath := filepath.Join(
			outputDir,
			fmt.Sprintf(
				"%s-%s.%d",
				config.BinaryName,
				name,
				config.Section,
			),
		)

		file, err := os.Create(pagePath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", pagePath, err)
		}

		if err := generateCommandPage(file, config, name, cmd); err != nil {
			file.Close()
			return fmt.Errorf("generating %s: %w", pagePath, err)
		}

		file.Close()
	}

	return nil
}

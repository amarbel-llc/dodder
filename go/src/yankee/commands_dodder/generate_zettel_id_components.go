package commands_dodder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
)

func init() {
	utility.AddCmd(
		"generate-zettel-id-components",
		&GenerateZettelIdComponents{},
	)
}

type GenerateZettelIdComponents struct{}

func (cmd GenerateZettelIdComponents) Run(req command.Request) {
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

	for _, component := range unicorn.ExtractUniqueComponents(lines) {
		fmt.Println(component)
	}
}

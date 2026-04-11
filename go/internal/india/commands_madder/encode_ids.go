package commands_madder

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func init() {
	utility.AddCmd("encode-ids", &EncodeIds{})
}

type EncodeIds struct{}

var _ command.CommandWithArgs = (*EncodeIds)(nil)

func (cmd EncodeIds) GetDescription() command.Description {
	return command.Description{
		Short: "convert hex digests to native markl IDs",
		Long: "Read hex-encoded digests from stdin (one per line) and " +
			"convert each to the native blech32-encoded markl ID format " +
			"with the specified hash type prefix. Useful for migrating " +
			"from external tools that produce hex digests.",
	}
}

func (cmd EncodeIds) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{
			Args: []command.Arg{
				{
					Name:        "hash-type",
					Description: "hash algorithm (e.g. sha256, blake2b256)",
					Required:    true,
				},
			},
		},
	}
}

func (cmd EncodeIds) Run(req command.Request) {
	hashType := req.PopArg("hash-type")

	if _, err := markl.GetFormatHashOrError(hashType); err != nil {
		errors.ContextCancelWithError(req, err)
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		id, repool := markl.GetId()

		if err := markl.SetHexBytes(hashType, id, []byte(line)); err != nil {
			repool()
			errors.ContextCancelWithError(req, err)
		}

		fmt.Println(id.String())
		repool()
	}

	if err := scanner.Err(); err != nil {
		errors.ContextCancelWithError(req, err)
	}
}

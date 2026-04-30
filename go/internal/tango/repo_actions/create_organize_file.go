package repo_actions

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/lima/orgie"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO support using query results for organize population
type CreateOrganizeFile struct {
	*repo
	orgie.Options
}

func (cmd CreateOrganizeFile) RunAndWrite(
	writer io.Writer,
) (results *orgie.Text, err error) {
	if results, err = cmd.Run(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	if _, err = results.WriteTo(writer); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}

func (cmd CreateOrganizeFile) Run() (results *orgie.Text, err error) {
	count := cmd.Options.Skus.Len()

	if cmd.Options.Limit == 0 && count > 30 && !cmd.GetCLIConfig().IsDryRun() {
		if !cmd.Confirm(
			fmt.Sprintf(
				"a large number (%d) of objects would be edited in organize. continue to organize?",
				count,
			),
			"",
		) {
			err = errors.Err499ClientClosedRequest
			return results, err
		}
	}

	if results, err = orgie.New(cmd.Options); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}

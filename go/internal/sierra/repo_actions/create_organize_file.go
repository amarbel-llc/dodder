package repo_actions

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
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

	// dodder#374(b): every generated organize document gets `_base`
	// pinned, mandatory with no legacy mode -- all three modes
	// (interactive, commit-directly, output-only) funnel through here.
	if err = WriteOrganizeBaseAndActivate(
		cmd.repo,
		results,
		cmd.Options.GroupingTags,
	); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}

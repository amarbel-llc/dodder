package files

import (
	"os/exec"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

func OpenFiles(p ...string) (err error) {
	cmd := exec.Command("open", p...)

	if err = cmd.Run(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

package user_ops

import (
	"fmt"
	"os/exec"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/google/shlex"
)

// TODO move to store_fs
type EachBlob struct {
	*local_working_copy.Repo
	Utility string
}

func (c EachBlob) Run(
	skus sku.SkuTypeSet,
) (err error) {
	if skus.Len() == 0 {
		return err
	}

	var blobPaths []string

	for checkedOut := range skus.All() {
		var fds *sku.FSItem

		if fds, err = c.GetEnvWorkspace().GetStoreFS().ReadFSItemFromExternal(
			checkedOut.GetSkuExternal(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		blobPaths = append(blobPaths, fds.Blob.GetPath())
	}

	v := fmt.Sprintf("running utility: %q", c.Utility)

	if err = c.PrinterHeader()(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var args []string

	if args, err = shlex.Split(c.Utility); err != nil {
		err = errors.Wrap(err)
		return err
	}

	args = append(args, blobPaths...)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = c.GetOutFile()
	cmd.Stdin = c.GetInFile()
	cmd.Stderr = c.GetErrFile()

	if err = cmd.Run(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

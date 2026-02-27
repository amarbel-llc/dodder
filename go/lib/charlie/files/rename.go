package files

import (
	"os"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

func Rename(src, dst string) (err error) {
	if err = os.Rename(src, dst); err != nil {
		err = errors.Wrapf(err, "Src: %q, Dst: %q", src, dst)
		return err
	}

	return err
}

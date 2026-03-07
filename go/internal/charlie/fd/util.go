package fd

import (
	"fmt"

	lib_fd "code.linenisgreat.com/dodder/go/lib/charlie/fd"
)

func Base(p string) string { return lib_fd.Base(p) }

func Dir(p string) string { return lib_fd.Dir(p) }

func DirBaseOnly(p string) string { return lib_fd.DirBaseOnly(p) }

func Ext(p string) string { return lib_fd.Ext(p) }

func ExtSansDot(p string) string { return lib_fd.ExtSansDot(p) }

func FileNameSansExt(p string) string { return lib_fd.FileNameSansExt(p) }

func FileNameSansExtRelTo(p, d string) (string, error) {
	return lib_fd.FileNameSansExtRelTo(p, d)
}

func ZettelId(p string) string {
	return fmt.Sprintf("%s/%s", DirBaseOnly(p), FileNameSansExt(p))
}

func FsRootDir() string { return lib_fd.FsRootDir() }

func DoesDirectoryContainPath(dir, path string) bool {
	return lib_fd.DoesDirectoryContainPath(dir, path)
}

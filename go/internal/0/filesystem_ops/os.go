package filesystem_ops

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

var ErrMergeConflict = errors.New("merge conflict")

type OsFilesystemOps struct {
	cwd string
}

func MakeOsFilesystemOps(cwd string) *OsFilesystemOps {
	return &OsFilesystemOps{cwd: cwd}
}

func (o *OsFilesystemOps) Open(
	path string,
	mode OpenMode,
) (io.ReadCloser, error) {
	switch mode {
	case OpenModeExclusive:
		return os.OpenFile(path, os.O_RDONLY|os.O_EXCL, 0o666)
	default:
		return os.Open(path)
	}
}

func (o *OsFilesystemOps) Create(
	path string,
	mode CreateMode,
) (io.WriteCloser, error) {
	switch mode {
	default:
		return os.OpenFile(
			path,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0o666,
		)
	}
}

func (o *OsFilesystemOps) CreateTemp(
	dir, pattern string,
) (string, io.WriteCloser, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", nil, err
	}

	return f.Name(), f, nil
}

func (o *OsFilesystemOps) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

func (o *OsFilesystemOps) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (o *OsFilesystemOps) Remove(path string) error {
	return os.RemoveAll(path)
}

func (o *OsFilesystemOps) GetCwd() string {
	return o.cwd
}

func (o *OsFilesystemOps) Rel(path string) (string, error) {
	return filepath.Rel(o.cwd, path)
}

func (o *OsFilesystemOps) Lstat(path string) (fs.FileInfo, error) {
	return os.Lstat(path)
}

func (o *OsFilesystemOps) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (o *OsFilesystemOps) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (o *OsFilesystemOps) Merge(
	base, current, other io.Reader,
) (io.ReadCloser, error) {
	basePath, err := writeReaderToTempFile(base)
	if err != nil {
		return nil, err
	}
	defer os.Remove(basePath)

	currentPath, err := writeReaderToTempFile(current)
	if err != nil {
		return nil, err
	}
	defer os.Remove(currentPath)

	otherPath, err := writeReaderToTempFile(other)
	if err != nil {
		return nil, err
	}
	defer os.Remove(otherPath)

	resultFile, err := os.CreateTemp("", "merge-result-*")
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"git", "merge-file", "-p",
		"-L=local", "-L=base", "-L=remote",
		currentPath, basePath, otherPath,
	)

	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		resultFile.Close()
		os.Remove(resultFile.Name())
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		resultFile.Close()
		os.Remove(resultFile.Name())
		return nil, err
	}

	if _, err := io.Copy(resultFile, stdout); err != nil {
		cmd.Wait()
		resultFile.Close()
		os.Remove(resultFile.Name())
		return nil, err
	}

	waitErr := cmd.Wait()

	if _, err := resultFile.Seek(0, io.SeekStart); err != nil {
		resultFile.Close()
		os.Remove(resultFile.Name())
		return nil, err
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return resultFile, ErrMergeConflict
		}

		resultFile.Close()
		os.Remove(resultFile.Name())
		return nil, waitErr
	}

	return resultFile, nil
}

func writeReaderToTempFile(r io.Reader) (string, error) {
	f, err := os.CreateTemp("", "merge-input-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}

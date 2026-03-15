package filesystem_ops

import (
	"io"
	"io/fs"
)

type OpenMode uint8

const (
	OpenModeDefault   OpenMode = iota // shared read
	OpenModeExclusive                 // exclusive lock (for blob reads)
)

type CreateMode uint8

const (
	CreateModeTruncate CreateMode = iota // create or truncate
)

type V0 interface {
	Open(path string, mode OpenMode) (io.ReadCloser, error)
	Create(path string, mode CreateMode) (io.WriteCloser, error)
	CreateTemp(dir, pattern string) (path string, w io.WriteCloser, err error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Rename(oldpath, newpath string) error
	Remove(path string) error
	GetCwd() string
	Rel(path string) (string, error)
	Lstat(path string) (fs.FileInfo, error)
	EvalSymlinks(path string) (string, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	Merge(base, current, other io.Reader) (io.ReadCloser, error)
}

type VCurrent = V0

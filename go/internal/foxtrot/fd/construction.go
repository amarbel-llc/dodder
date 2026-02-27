package fd

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func MakeFromDirPath(
	path string,
) (fd *FD, err error) {
	fd = &FD{}
	fd.isDir = true
	fd.path = path

	return fd, err
}

func MakeFromPathAndDirEntry(
	path string,
	dirEntry fs.DirEntry,
	blobWriter domain_interfaces.BlobWriterFactory,
) (fd *FD, err error) {
	if path == "" {
		err = errors.ErrorWithStackf("nil file desriptor")
		return fd, err
	}

	if path == "." {
		err = errors.ErrorWithStackf("'.' not supported")
		return fd, err
	}

	var fi os.FileInfo

	if fi, err = dirEntry.Info(); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	if fd, err = MakeFromFileInfoWithDir(fi, filepath.Dir(path), blobWriter); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	return fd, err
}

func MakeFromPath(
	baseDir string,
	path string,
	blobWriter domain_interfaces.BlobWriterFactory,
) (fd *FD, err error) {
	if path == "" {
		err = errors.ErrorWithStackf("nil file desriptor")
		return fd, err
	}

	if path == "." {
		err = errors.ErrorWithStackf("'.' not supported")
		return fd, err
	}

	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(baseDir, path))
	}

	var fi os.FileInfo

	if fi, err = os.Stat(path); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	if fd, err = MakeFromFileInfoWithDir(
		fi,
		filepath.Dir(path),
		blobWriter,
	); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	return fd, err
}

func MakeFromFileInfoWithDir(
	fileInfo os.FileInfo,
	dir string,
	blobStore domain_interfaces.BlobWriterFactory,
) (fd *FD, err error) {
	// TODO use pool
	fd = &FD{}

	if err = fd.SetFileInfoWithDir(fileInfo, dir); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	if fileInfo.IsDir() {
		return fd, err
	}

	// TODO eventually enforce requirement of blob writer factory
	if blobStore == nil {
		return fd, err
	}

	var file *os.File

	if file, err = files.OpenExclusiveReadOnly(fd.GetPath()); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	defer errors.DeferredCloser(&err, file)

	var writer domain_interfaces.BlobWriter

	if writer, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	defer errors.DeferredCloser(&err, writer)

	if _, err = io.Copy(writer, file); err != nil {
		err = errors.Wrap(err)
		return fd, err
	}

	markl.SetDigester(&fd.digest, writer)
	fd.state = StateStored

	return fd, err
}

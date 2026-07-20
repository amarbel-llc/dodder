package env_repo

import (
	"io"
	"os"
	"path/filepath"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/madder/go/pkgs/markl_io"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
)

// namedBlobWriter writes to a named (non-content-addressed) path
// with atomic-rename overwrite semantics. Used for dodder's
// named-blob caches (e.g. zettel id index bitset) where each
// flush replaces the previous content.
//
// This is intentionally separate from madder's blob_io.NewMover.
// madder's mover is purpose-built for content-addressed publishes:
// on os.Link ErrExist it treats the collision as benign per
// ADR 0003 and silently swallows the temp file. Reusing it for
// named writes that must overwrite produces a silent-data-loss
// bug — dodder's zettel_id_index Flush would never persist
// updates, causing the bitset to drift and re-allocate the same
// id twice on subsequent calls.
type namedBlobWriter struct {
	finalPath string

	tempFile *os.File
	digester mad_domain_interfaces.BlobWriter
	tee      io.Writer
}

// makeNamedBlobWriter creates a writer that buffers content into a
// temp file under tempFS and atomically renames it over finalPath
// on Close. hashType determines the digest reported by GetMarklId.
func makeNamedBlobWriter(
	finalPath string,
	hashType mad_domain_interfaces.FormatHash,
	tempFS files.TemporaryFS,
) (writer *namedBlobWriter, err error) {
	writer = &namedBlobWriter{
		finalPath: finalPath,
	}

	if err = os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		err = errors.Wrap(err)
		return writer, err
	}

	if writer.tempFile, err = tempFS.FileTemp(); err != nil {
		err = errors.Wrap(err)
		return writer, err
	}

	hash, _ := hashType.GetHash() //repool:owned
	writer.digester = markl_io.MakeWriter(hash, nil)
	writer.tee = io.MultiWriter(writer.tempFile, writer.digester)

	return writer, err
}

func (w *namedBlobWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if n, err = io.Copy(w.tee, r); err != nil {
		err = errors.Wrap(err)
		return n, err
	}
	return n, err
}

func (w *namedBlobWriter) Write(p []byte) (n int, err error) {
	return w.tee.Write(p)
}

func (w *namedBlobWriter) Close() (err error) {
	if w.tempFile == nil {
		err = errors.ErrorWithStackf("named blob writer already closed")
		return err
	}

	tempPath := w.tempFile.Name()

	if err = w.tempFile.Sync(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = w.tempFile.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	w.tempFile = nil

	// os.Rename overwrites an existing destination atomically on POSIX
	// when both paths are on the same filesystem. dodder's named blob
	// caches require this — each flush replaces the previous bytes.
	if err = os.Rename(tempPath, w.finalPath); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (w *namedBlobWriter) GetMarklId() mad_domain_interfaces.MarklId {
	return w.digester.GetMarklId()
}

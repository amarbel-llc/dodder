package domain_interfaces

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type (
	BlobIOWrapper interface {
		GetBlobEncryption() MarklId
		GetBlobCompression() interfaces.IOWrapper
	}

	BlobIOWrapperGetter interface {
		GetBlobIOWrapper() BlobIOWrapper
	}

	ReadAtSeeker interface {
		io.ReaderAt
		io.Seeker
	}

	BlobReader interface {
		io.WriterTo
		io.ReadCloser

		ReadAtSeeker
		MarklIdGetter
	}

	BlobWriter interface {
		io.ReaderFrom
		io.WriteCloser
		MarklIdGetter
	}

	BlobReaderFactory interface {
		MakeBlobReader(MarklId) (BlobReader, error)
	}

	BlobWriterFactory interface {
		MakeBlobWriter(FormatHash) (BlobWriter, error)
	}

	BlobAccess interface {
		HasBlob(MarklId) bool
		BlobReaderFactory
		BlobWriterFactory
	}

	NamedBlobAccess interface {
		MakeNamedBlobReader(string) (BlobReader, error)
		MakeNamedBlobWriter(string) (BlobWriter, error)
	}

	BlobStore interface {
		BlobAccess
		BlobIOWrapperGetter

		GetBlobStoreDescription() string
		GetDefaultHashType() FormatHash
		AllBlobs() interfaces.SeqError[MarklId]
	}

	// Blobs represent persisted files, like blobs in Git. Blobs are used by
	// Zettels, types, tags, config, and inventory lists.
	BlobPool[BLOB any] interface {
		GetBlob(MarklId) (BLOB, interfaces.FuncRepool, error)
	}

	Format[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]] interface {
		SavedBlobFormatter
		interfaces.CoderReadWriter[BLOB_PTR]
	}

	TypedStore[
		BLOB any,
		BLOB_PTR interfaces.Ptr[BLOB],
	] interface {
		// TODO remove and replace with two-step process
		SaveBlobText(BLOB_PTR) (MarklId, int64, error)
		Format[BLOB, BLOB_PTR]
		// TODO remove
		BlobPool[BLOB_PTR]
	}

	SavedBlobFormatter interface {
		FormatSavedBlob(io.Writer, MarklId) (int64, error)
	}

	BlobForeignDigestAdder interface {
		AddForeignBlobDigestForNativeDigest(foreign, native MarklId) error
	}
)

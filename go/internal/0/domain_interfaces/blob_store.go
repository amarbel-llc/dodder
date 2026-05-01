package domain_interfaces

import (
	madder_di "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	BlobIOWrapper                                       = madder_di.BlobIOWrapper
	BlobIOWrapperGetter                                 = madder_di.BlobIOWrapperGetter
	ReadAtSeeker                                        = madder_di.ReadAtSeeker
	BlobReader                                          = madder_di.BlobReader
	BlobWriter                                          = madder_di.BlobWriter
	BlobReaderFactory                                   = madder_di.BlobReaderFactory
	BlobWriterFactory                                   = madder_di.BlobWriterFactory
	BlobAccess                                          = madder_di.BlobAccess
	NamedBlobAccess                                     = madder_di.NamedBlobAccess
	BlobPool[BLOB any]                                  = madder_di.BlobPool[BLOB]
	Format[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]]     = madder_di.Format[BLOB, BLOB_PTR]
	TypedStore[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]] = madder_di.TypedStore[BLOB, BLOB_PTR]
	SavedBlobFormatter                                  = madder_di.SavedBlobFormatter
	BlobForeignDigestAdder                              = madder_di.BlobForeignDigestAdder
	BlobStore                                           = madder_di.BlobStore
	BlobStoreConfig                                     = madder_di.BlobStoreConfig
)

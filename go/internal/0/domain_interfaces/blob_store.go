package domain_interfaces

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	BlobIOWrapper                                       = mad_domain_interfaces.BlobIOWrapper
	BlobIOWrapperGetter                                 = mad_domain_interfaces.BlobIOWrapperGetter
	ReadAtSeeker                                        = mad_domain_interfaces.ReadAtSeeker
	BlobReader                                          = mad_domain_interfaces.BlobReader
	BlobWriter                                          = mad_domain_interfaces.BlobWriter
	BlobReaderFactory                                   = mad_domain_interfaces.BlobReaderFactory
	BlobWriterFactory                                   = mad_domain_interfaces.BlobWriterFactory
	BlobAccess                                          = mad_domain_interfaces.BlobAccess
	NamedBlobAccess                                     = mad_domain_interfaces.NamedBlobAccess
	BlobPool[BLOB any]                                  = mad_domain_interfaces.BlobPool[BLOB]
	Format[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]]     = mad_domain_interfaces.Format[BLOB, BLOB_PTR]
	TypedStore[BLOB any, BLOB_PTR interfaces.Ptr[BLOB]] = mad_domain_interfaces.TypedStore[BLOB, BLOB_PTR]
	SavedBlobFormatter                                  = mad_domain_interfaces.SavedBlobFormatter
	BlobForeignDigestAdder                              = mad_domain_interfaces.BlobForeignDigestAdder
	BlobStore                                           = mad_domain_interfaces.BlobStore
	BlobStoreConfig                                     = mad_domain_interfaces.BlobStoreConfig
	BlobWriteObserver                                   = mad_domain_interfaces.BlobWriteObserver
	BlobWriteEvent                                      = mad_domain_interfaces.BlobWriteEvent
	BlobWriteOp                                         = mad_domain_interfaces.BlobWriteOp
)

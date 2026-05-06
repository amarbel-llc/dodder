// Package env_dir is a thin forwarder over madder's pkgs/env_dir
// and pkgs/blob_io.
package env_dir

import (
	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	mad_env_dir "github.com/amarbel-llc/madder/go/pkgs/env_dir"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
)

const (
	EnvDir               = dodder_env.EnvDir
	XDGUtilityNameDodder = dodder_env.XDGUtilityName
)

type (
	Env          = mad_env_dir.Env
	RelativePath = mad_env_dir.RelativePath
	TemporaryFS  = mad_env_dir.TemporaryFS

	Config               = mad_blob_io.Config
	MoveOptions          = mad_blob_io.MoveOptions
	ErrBlobAlreadyExists = mad_blob_io.ErrBlobAlreadyExists
	ErrBlobMissing       = mad_blob_io.ErrBlobMissing
)

var (
	DefaultConfig                                = mad_blob_io.DefaultConfig
	MakeConfig                                   = mad_blob_io.MakeConfig
	NewReader                                    = mad_blob_io.NewReader
	NewFileReaderOrErrNotExist                   = mad_blob_io.NewFileReaderOrErrNotExist
	NewNopReader                                 = mad_blob_io.NewNopReader
	NewWriter                                    = mad_blob_io.NewWriter
	NewMover                                     = mad_blob_io.NewMover
	IsErrBlobAlreadyExists                       = mad_blob_io.IsErrBlobAlreadyExists
	IsErrBlobMissing                             = mad_blob_io.IsErrBlobMissing
	MakeHashBucketPath                           = mad_blob_io.MakeHashBucketPath
	MakeHashBucketPathFromMerkleId               = mad_blob_io.MakeHashBucketPathFromMerkleId
	MakeHashBucketPathJoinFunc                   = mad_blob_io.MakeHashBucketPathJoinFunc
	MakeDirIfNecessary                           = mad_blob_io.MakeDirIfNecessary
	MakeDirIfNecessaryForStringerWithHeadAndTail = mad_blob_io.MakeDirIfNecessaryForStringerWithHeadAndTail
	PathFromHeadAndTail                          = mad_blob_io.PathFromHeadAndTail
)

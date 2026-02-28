package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func MakeHashBucketPathFromMerkleId(
	id domain_interfaces.MarklId,
	buckets []int,
	multiHash bool,
	pathComponents ...string,
) string {
	if multiHash {
		pathComponents = append(
			pathComponents,
			id.GetMarklFormat().GetMarklFormatId(),
		)
	}

	return files.MakeHashBucketPath(
		[]byte(markl.FormatBytesAsHex(id)),
		buckets,
		pathComponents...,
	)
}

var MakeHashBucketPath = files.MakeHashBucketPath

var PathFromHeadAndTail = files.PathFromHeadAndTail

var MakeHashBucketPathJoinFunc = files.MakeHashBucketPathJoinFunc

var MakeDirIfNecessary = files.MakeDirIfNecessary

var MakeDirIfNecessaryForStringerWithHeadAndTail = files.MakeDirIfNecessaryForStringerWithHeadAndTail

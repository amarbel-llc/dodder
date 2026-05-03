package env_dir

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
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

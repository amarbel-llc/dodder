package page_id

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

// TODO rename to Id
type PageId struct {
	Index  uint8
	Dir    string
	Prefix string
}

func PageIdFromPath(n uint8, path string) PageId {
	dir, file := filepath.Split(path)
	return PageId{
		Dir:    dir,
		Prefix: file,
		Index:  n,
	}
}

func (id PageId) String() string {
	return fmt.Sprintf("%d", id.Index)
}

func (id *PageId) Path() string {
	return filepath.Join(id.Dir, fmt.Sprintf("%s-%x", id.Prefix, id.Index))
}

func PageIndexForString(
	width uint8,
	value string,
	hashType domain_interfaces.FormatHash,
) (n uint8, err error) {
	digest, repool := hashType.GetMarklIdForString(value)
	defer repool()

	if n, err = PageIndexForDigest(width, digest); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func PageIndexForDigest(
	width uint8,
	digest domain_interfaces.MarklId,
) (n uint8, err error) {
	var n1 int64

	if err = markl.AssertIdIsNotNull(digest); err != nil {
		panic(err)
	}

	if n, err = PageIndexForDigestString(
		width,
		markl.FormatBytesAsHex(digest),
	); err != nil {
		err = errors.ErrorWithStackf("page out of bounds: %d", n1)
		return n, err
	}

	return n, err
}

func PageIndexForDigestString(
	width uint8,
	digestString string,
) (n uint8, err error) {
	var n1 int64

	bucketIndexString := digestString[:width]

	if n1, err = strconv.ParseInt(bucketIndexString, 16, 64); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if n1 > math.MaxUint8 {
		err = errors.ErrorWithStackf("page out of bounds: %d", n1)
		return n, err
	}

	n = uint8(n1)

	return n, err
}

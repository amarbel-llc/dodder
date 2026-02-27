package object_probe_index

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func (page *page) seekToFirstBinarySearch(
	expected domain_interfaces.MarklId,
) (mid int64, err error) {
	errors.PanicIfError(markl.AssertIdIsNotNull(expected))

	errors.PanicIfError(
		markl.MakeErrWrongType(
			page.hashType.GetMarklFormatId(),
			expected.GetMarklFormat().GetMarklFormatId(),
		),
	)

	if page.file == nil {
		err = errors.MakeErrNotFoundString(
			"fd nil: " + expected.StringWithFormat(),
		)
		return mid, err
	}

	var low, hi int64

	var rowCount int64

	if rowCount, err = page.GetRowCount(); err != nil {
		err = errors.Wrap(err)
		return mid, err
	}

	hi = rowCount
	loops := 0

	for low <= hi {
		loops++
		mid = (hi + low) / 2

		// var loc int64

		cmp := markl.CompareToReaderAt(
			page.file,
			mid*int64(page.rowWidth),
			expected,
		)

		switch cmp {
		case -1:
			if low == hi-1 {
				low = hi
			} else {
				hi = mid - 1
			}

		case 0:
			// found
			return mid, err

		case 1:
			low = mid + 1

		default:
			panic("not possible")
		}
	}

	err = errors.MakeErrNotFoundString(
		fmt.Sprintf("%d: %s", loops, expected.StringWithFormat()),
	)

	return mid, err
}

func (page *page) seekToFirstLinearSearch(
	expected domain_interfaces.MarklId,
) (loc int64, err error) {
	errors.PanicIfError(markl.AssertIdIsNotNull(expected))

	errors.PanicIfError(
		markl.MakeErrWrongType(
			page.hashType.GetMarklFormatId(),
			expected.GetMarklFormat().GetMarklFormatId(),
		),
	)

	if page.file == nil {
		err = errors.MakeErrNotFoundString(
			"fd nil: " + expected.StringWithFormat(),
		)
		return loc, err
	}

	var rowCount int64

	if rowCount, err = page.GetRowCount(); err != nil {
		err = errors.Wrap(err)
		return loc, err
	}

	page.bufferedReader.Reset(page.file)

	for loc = int64(0); loc <= rowCount; loc++ {
		// var loc int64

		if markl.CompareToReader(&page.bufferedReader, expected) == 0 {
			return loc, err
		}
	}

	err = errors.MakeErrNotFoundString(expected.StringWithFormat())

	return loc, err
}

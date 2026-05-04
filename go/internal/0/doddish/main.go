package doddish

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
)

func ScanExactlyOneSeqWithDotAllowedInIdenfierFromString(
	value string,
) (seq Seq, err Error) {
	reader, repool := pool.GetStringReader(value)
	defer repool()

	var boxScanner Scanner
	boxScanner.Reset(reader)

	if seq, err = boxScanner.ScanDotAllowedInIdentifiersOrError(); err != nil {
		return seq, err
	}

	return seq, err
}

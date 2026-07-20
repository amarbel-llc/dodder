package doddish

import (
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
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

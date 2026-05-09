package store_config

import (
	"bufio"
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestListCoderRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)

	ta, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	t.AssertNoError(ta.GetObjectIdMutable().Set("test-tag"))

	var buf bytes.Buffer
	var coder stream_index.ListCoder

	writer := bufio.NewWriter(&buf)

	_, err := coder.EncodeTo(ta, writer)
	t.AssertNoError(err)

	t.AssertNoError(writer.Flush())

	reader := bufio.NewReader(&buf)

	var actual sku.Transacted

	_, err = coder.DecodeFrom(&actual, reader)
	t.AssertNoError(err)

	t.AssertEqual(ta.GetObjectId().String(), actual.GetObjectId().String())
}

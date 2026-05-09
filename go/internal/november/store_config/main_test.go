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

	if err := ta.GetObjectIdMutable().Set("test-tag"); err != nil {
		t.Fatalf("failed to set object id: %s", err)
	}

	var buf bytes.Buffer
	var coder stream_index.ListCoder

	writer := bufio.NewWriter(&buf)

	if _, err := coder.EncodeTo(ta, writer); err != nil {
		t.Fatalf("failed to encode: %s", err)
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("failed to flush: %s", err)
	}

	reader := bufio.NewReader(&buf)

	var actual sku.Transacted

	if _, err := coder.DecodeFrom(&actual, reader); err != nil {
		t.Fatalf("failed to decode: %s", err)
	}

	t.AssertEqual(ta.GetObjectId().String(), actual.GetObjectId().String())
}

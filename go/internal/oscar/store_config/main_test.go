package store_config

import (
	"bufio"
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/stream_index"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestListCoderRoundTrip(t1 *testing.T) {
	t := ui.T{T: t1}

	ta, _ := sku.GetTransactedPool().GetWithRepool()

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

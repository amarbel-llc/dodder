package store_config

import (
	"bytes"
	"encoding/gob"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

// TODO remove
func TestGob(t1 *testing.T) {
	t := ui.T{T: t1}

	ta, _ := sku.GetTransactedPool().GetWithRepool()

	if err := ta.GetObjectIdMutable().Set("test-tag"); err != nil {
		t.Fatalf("failed to set object id: %s", err)
	}

	var b bytes.Buffer

	enc := gob.NewEncoder(&b)

	if err := enc.Encode(ta); err != nil {
		t.Fatalf("failed to encode config: %s", err)
	}

	dec := gob.NewDecoder(&b)

	var actual sku.Transacted

	if err := dec.Decode(&actual); err != nil {
		t.Fatalf("failed to decode config: %s", err)
	}

	t.AssertEqual(ta.GetObjectId().String(), actual.GetObjectId().String())
}

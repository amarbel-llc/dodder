package store_config

import (
	"bytes"
	"encoding/gob"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

// TODO remove
func TestGob(t1 *testing.T) {
	t := ui.T{T: t1}

	ta, _ := sku.GetTransactedPool().GetWithRepool()

	if err := ta.ObjectId.Set("test-tag"); err != nil {
		t.Fatalf("failed to set object id: %w", err)
	}

	var b bytes.Buffer

	enc := gob.NewEncoder(&b)

	if err := enc.Encode(ta); err != nil {
		t.Fatalf("failed to encode config: %w", err)
	}

	dec := gob.NewDecoder(&b)

	var actual sku.Transacted

	if err := dec.Decode(&actual); err != nil {
		t.Fatalf("failed to decode config: %w", err)
	}

	t.AssertEqual(ta.ObjectId.String(), actual.ObjectId.String())
}

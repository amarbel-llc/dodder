package sku

import (
	"bytes"
	"encoding/gob"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

func TestGob(t1 *testing.T) {
	t := ui.T{T: t1}

	type testType = Transacted

	var expected testType

	if err := expected.ObjectId.Set("test-tag"); err != nil {
		t.Fatalf("failed to set object id: %w", err)
	}

	var b bytes.Buffer

	enc := gob.NewEncoder(&b)

	if err := enc.Encode(&expected); err != nil {
		t.Fatalf("failed to encode config: %w", err)
	}

	dec := gob.NewDecoder(&b)

	var actual testType

	if err := dec.Decode(&actual); err != nil {
		t.Fatalf("failed to decode config: %w", err)
	}

	t.AssertEqual(expected.ObjectId.String(), actual.ObjectId.String())
}

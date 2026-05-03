//go:build test

package sku_fmt

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type failingTypeBlobStore struct {
	err error
}

func (s failingTypeBlobStore) ParseTypedBlob(
	domain_interfaces.ObjectId,
	domain_interfaces.MarklId,
) (type_blobs.Blob, interfaces.FuncRepool, int64, error) {
	return nil, func() {}, 0, s.err
}

func TestFormatReturnsErrorOnParseBlobFailure(t *testing.T) {
	expectedErr := errors.Errorf("blob parse failed")

	typeObject, typeObjectRepool := sku.GetTransactedPool().GetWithRepool()
	defer typeObjectRepool()

	format := MakeFormatterTypeFormatterUTIGroups(
		func(objects.TypeLock) (*sku.Transacted, error) {
			return typeObject, nil
		},
		failingTypeBlobStore{err: expectedErr},
	)

	object, objectRepool := sku.GetTransactedPool().GetWithRepool()
	defer objectRepool()

	var buf bytes.Buffer

	_, err := format.Format(&buf, object)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

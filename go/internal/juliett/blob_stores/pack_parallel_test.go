//go:build test && debug

package blob_stores

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
)

func TestCollectBlobMetasParallelBasic(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	testData := map[string][]byte{
		"blob1": []byte("parallel test blob one"),
		"blob2": []byte("parallel test blob two"),
		"blob3": []byte("parallel test blob three"),
	}

	var allIds []domain_interfaces.MarklId
	blobData := make(map[string][]byte)

	for _, data := range testData {
		rawHash := sha256.Sum256(data)
		id, repool := hashFormat.GetBlobIdForHexString(
			hex.EncodeToString(rawHash[:]),
		)
		defer repool()

		allIds = append(allIds, id)
		blobData[id.String()] = data
	}

	stub := &stubBlobStore{
		allBlobIds: allIds,
		blobData:   blobData,
	}

	sizeFn := func(id domain_interfaces.MarklId) (uint64, error) {
		if data, ok := blobData[id.String()]; ok {
			return uint64(len(data)), nil
		}
		return 0, nil
	}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		make(map[string]bool),
		PackOptions{},
		sizeFn,
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if len(metas) != 3 {
		t.Fatalf("expected 3 metas, got %d", len(metas))
	}

	// Verify sorted by digest.
	for i := 1; i < len(metas); i++ {
		if bytes.Compare(metas[i-1].digest, metas[i].digest) >= 0 {
			t.Errorf("metas not sorted at index %d", i)
		}
	}

	// Verify sizes are non-zero.
	for i, m := range metas {
		if m.size == 0 {
			t.Errorf("meta %d has zero size", i)
		}
	}
}

func TestCollectBlobMetasParallelSkipsArchived(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	testData1 := []byte("archived blob data")
	testData2 := []byte("loose only blob data")

	rawHash1 := sha256.Sum256(testData1)
	rawHash2 := sha256.Sum256(testData2)

	id1, repool1 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash1[:]),
	)
	defer repool1()

	id2, repool2 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash2[:]),
	)
	defer repool2()

	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{id1, id2},
		blobData: map[string][]byte{
			id1.String(): testData1,
			id2.String(): testData2,
		},
	}

	// Mark id1 as already archived.
	indexPresence := map[string]bool{
		id1.String(): true,
	}

	sizeFn := func(id domain_interfaces.MarklId) (uint64, error) {
		if data, ok := stub.blobData[id.String()]; ok {
			return uint64(len(data)), nil
		}
		return 0, nil
	}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		indexPresence,
		PackOptions{},
		sizeFn,
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if len(metas) != 1 {
		t.Fatalf("expected 1 meta (archived blob skipped), got %d", len(metas))
	}
}

func TestCollectBlobMetasParallelEmpty(t *testing.T) {
	stub := &stubBlobStore{}

	metas, err := collectBlobMetasParallel(
		nil,
		nil,
		stub,
		make(map[string]bool),
		PackOptions{},
		func(domain_interfaces.MarklId) (uint64, error) { return 0, nil },
	)
	if err != nil {
		t.Fatalf("collectBlobMetasParallel: %v", err)
	}

	if metas != nil {
		t.Fatalf("expected nil metas for empty store, got %d", len(metas))
	}
}

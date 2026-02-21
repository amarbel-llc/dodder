//go:build test && debug

package inventory_archive

import (
	"bytes"
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

type testBlobReader struct {
	*bytes.Reader
}

func makeTestBlobReader(data []byte) *testBlobReader {
	return &testBlobReader{Reader: bytes.NewReader(data)}
}

func (r *testBlobReader) Close() error { return nil }

func (r *testBlobReader) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, r.Reader)
}

func (r *testBlobReader) GetMarklId() domain_interfaces.MarklId { return nil }

func TestBsdiffRoundTrip(t *testing.T) {
	base := []byte("the quick brown fox jumps over the lazy dog")
	target := []byte("the quick brown cat jumps over the lazy dog")

	alg := &Bsdiff{}

	var deltaBuf bytes.Buffer

	baseReader := makeTestBlobReader(base)

	err := alg.Compute(
		baseReader,
		int64(len(base)),
		bytes.NewReader(target),
		&deltaBuf,
	)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if deltaBuf.Len() == 0 {
		t.Fatal("expected non-empty delta")
	}

	var reconstructed bytes.Buffer

	baseReader2 := makeTestBlobReader(base)
	err = alg.Apply(
		baseReader2,
		int64(len(base)),
		&deltaBuf,
		&reconstructed,
	)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if !bytes.Equal(reconstructed.Bytes(), target) {
		t.Errorf(
			"reconstructed data mismatch: got %q, want %q",
			reconstructed.Bytes(),
			target,
		)
	}
}

func TestBsdiffIdenticalBlobs(t *testing.T) {
	data := []byte("identical content")

	alg := &Bsdiff{}

	var deltaBuf bytes.Buffer

	baseReader := makeTestBlobReader(data)
	err := alg.Compute(
		baseReader,
		int64(len(data)),
		bytes.NewReader(data),
		&deltaBuf,
	)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	var reconstructed bytes.Buffer

	baseReader2 := makeTestBlobReader(data)
	err = alg.Apply(
		baseReader2,
		int64(len(data)),
		&deltaBuf,
		&reconstructed,
	)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if !bytes.Equal(reconstructed.Bytes(), data) {
		t.Errorf("reconstructed data mismatch for identical blobs")
	}
}

func TestBsdiffId(t *testing.T) {
	alg := &Bsdiff{}
	if alg.Id() != DeltaAlgorithmByteBsdiff {
		t.Errorf("expected id %d, got %d", DeltaAlgorithmByteBsdiff, alg.Id())
	}
}

func TestBsdiffRegistered(t *testing.T) {
	alg, err := DeltaAlgorithmForByte(DeltaAlgorithmByteBsdiff)
	if err != nil {
		t.Fatalf("bsdiff should be registered: %v", err)
	}

	if alg.Id() != DeltaAlgorithmByteBsdiff {
		t.Errorf("expected id %d, got %d", DeltaAlgorithmByteBsdiff, alg.Id())
	}
}

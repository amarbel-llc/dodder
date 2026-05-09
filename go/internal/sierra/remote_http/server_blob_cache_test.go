package remote_http

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

// stubBlobStoreForCache satisfies mad_domain_interfaces.BlobStore for cache tests.
// Only AllBlobs() is implemented; other methods panic if invoked, which keeps
// the stub minimal and surfaces unexpected use during tests.
type stubBlobStoreForCache struct {
	mad_domain_interfaces.BlobStore

	ids       []mad_domain_interfaces.MarklId
	yieldErr  error
	calls     atomic.Int64
	delayInit chan struct{}
}

func (s *stubBlobStoreForCache) AllBlobs() interfaces.SeqError[mad_domain_interfaces.MarklId] {
	return func(yield func(mad_domain_interfaces.MarklId, error) bool) {
		s.calls.Add(1)

		if s.delayInit != nil {
			<-s.delayInit
		}

		for _, id := range s.ids {
			if !yield(id, nil) {
				return
			}
		}

		if s.yieldErr != nil {
			var zero mad_domain_interfaces.MarklId
			yield(zero, s.yieldErr)
		}
	}
}

func makeTestMarklIds(t *ui.T, hexes ...string) ([]mad_domain_interfaces.MarklId, func()) {
	t.Helper()

	ids := make([]mad_domain_interfaces.MarklId, len(hexes))
	repools := make([]func(), 0, len(hexes))

	for i, h := range hexes {
		id, repool := markl.FormatHashSha256.GetBlobIdForHexString(h)
		ids[i] = id
		repools = append(repools, repool)
	}

	cleanup := func() {
		for _, r := range repools {
			r()
		}
	}

	return ids, cleanup
}

func TestServerBlobCacheHasBlobReturnsTrueForKnown(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexA := "0000000000000000000000000000000000000000000000000000000000000001"
	hexB := "0000000000000000000000000000000000000000000000000000000000000002"

	ids, cleanup := makeTestMarklIds(&t, hexA, hexB)
	defer cleanup()

	stub := &stubBlobStoreForCache{ids: ids}
	cache := &serverBlobCache{localBlobStore: stub}

	for _, id := range ids {
		ok, err := cache.HasBlob(id)
		t.AssertNoError(err)
		if !ok {
			t.Fatalf("HasBlob(%v) = false, want true", id)
		}
	}
}

func TestServerBlobCacheHasBlobReturnsFalseForUnknown(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexKnown := "0000000000000000000000000000000000000000000000000000000000000001"
	hexUnknown := "00000000000000000000000000000000000000000000000000000000000000ff"

	ids, cleanup := makeTestMarklIds(&t, hexKnown, hexUnknown)
	defer cleanup()

	stub := &stubBlobStoreForCache{ids: ids[:1]}
	cache := &serverBlobCache{localBlobStore: stub}

	ok, err := cache.HasBlob(ids[1])
	t.AssertNoError(err)
	t.AssertFalse(ok, "HasBlob(unknown) = true, want false")
}

func TestServerBlobCacheInitRunsExactlyOnceConcurrent(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexes := []string{
		"0000000000000000000000000000000000000000000000000000000000000001",
		"0000000000000000000000000000000000000000000000000000000000000002",
		"0000000000000000000000000000000000000000000000000000000000000003",
	}
	ids, cleanup := makeTestMarklIds(&t, hexes...)
	defer cleanup()

	gate := make(chan struct{})
	stub := &stubBlobStoreForCache{
		ids:       ids,
		delayInit: gate,
	}
	cache := &serverBlobCache{localBlobStore: stub}

	const N = 32
	var wg sync.WaitGroup
	wg.Add(N)
	start := make(chan struct{})

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			ok, err := cache.HasBlob(ids[i%len(ids)])
			if err != nil {
				t1.Errorf("goroutine %d: unexpected err: %v", i, err)
			}
			if !ok {
				t1.Errorf("goroutine %d: HasBlob = false, want true", i)
			}
		}(i)
	}

	close(start)
	close(gate)
	wg.Wait()

	if got := stub.calls.Load(); got != 1 {
		t.Fatalf("expected AllBlobs to be called exactly once across %d concurrent HasBlob calls, got: %d", N, got)
	}
}

func TestServerBlobCacheMixedReadPatternsConsistent(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexes := []string{
		"00000000000000000000000000000000000000000000000000000000000000aa",
		"00000000000000000000000000000000000000000000000000000000000000bb",
		"00000000000000000000000000000000000000000000000000000000000000cc",
	}
	ids, cleanup := makeTestMarklIds(&t, hexes...)
	defer cleanup()

	hexUnknown := "00000000000000000000000000000000000000000000000000000000000000ff"
	unknownIds, cleanup2 := makeTestMarklIds(&t, hexUnknown)
	defer cleanup2()

	stub := &stubBlobStoreForCache{ids: ids}
	cache := &serverBlobCache{localBlobStore: stub}

	for round := 0; round < 5; round++ {
		for _, id := range ids {
			ok, err := cache.HasBlob(id)
			if err != nil {
				t.Fatalf("round %d: HasBlob(known) err: %v", round, err)
			}
			if !ok {
				t.Fatalf("round %d: HasBlob(known) = false", round)
			}
		}
		ok, err := cache.HasBlob(unknownIds[0])
		if err != nil {
			t.Fatalf("round %d: HasBlob(unknown) err: %v", round, err)
		}
		if ok {
			t.Fatalf("round %d: HasBlob(unknown) = true", round)
		}
	}

	if got := stub.calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 AllBlobs call across mixed reads, got: %d", got)
	}
}

func TestServerBlobCacheInstancesIndependent(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexA := "0000000000000000000000000000000000000000000000000000000000000001"
	hexB := "0000000000000000000000000000000000000000000000000000000000000002"
	idsA, cleanupA := makeTestMarklIds(&t, hexA)
	defer cleanupA()
	idsB, cleanupB := makeTestMarklIds(&t, hexB)
	defer cleanupB()

	stubA := &stubBlobStoreForCache{ids: idsA}
	stubB := &stubBlobStoreForCache{ids: idsB}
	cacheA := &serverBlobCache{localBlobStore: stubA}
	cacheB := &serverBlobCache{localBlobStore: stubB}

	okAA, err := cacheA.HasBlob(idsA[0])
	if err != nil || !okAA {
		t.Fatalf("cacheA HasBlob(A): ok=%v err=%v", okAA, err)
	}
	okAB, err := cacheA.HasBlob(idsB[0])
	if err != nil {
		t.Fatalf("cacheA HasBlob(B) err: %v", err)
	}
	if okAB {
		t.Fatalf("cacheA HasBlob(B) = true; should not see B's blobs")
	}

	okBB, err := cacheB.HasBlob(idsB[0])
	if err != nil || !okBB {
		t.Fatalf("cacheB HasBlob(B): ok=%v err=%v", okBB, err)
	}
	okBA, err := cacheB.HasBlob(idsA[0])
	if err != nil {
		t.Fatalf("cacheB HasBlob(A) err: %v", err)
	}
	if okBA {
		t.Fatalf("cacheB HasBlob(A) = true; should not see A's blobs")
	}
}

func TestServerBlobCachePopulateErrorPropagates(t1 *testing.T) {
	t := ui.MakeT(t1)

	hexA := "0000000000000000000000000000000000000000000000000000000000000001"
	ids, cleanup := makeTestMarklIds(&t, hexA)
	defer cleanup()

	wantErr := errors.New("simulated populate failure")
	stub := &stubBlobStoreForCache{
		ids:      ids,
		yieldErr: wantErr,
	}
	cache := &serverBlobCache{localBlobStore: stub}

	_, err := cache.HasBlob(ids[0])
	t.AssertError(err)
	if !errors.Is(err, wantErr) && !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("expected wrapped error to reference %q, got: %v", wantErr, err)
	}
}

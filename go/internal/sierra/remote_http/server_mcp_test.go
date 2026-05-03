package remote_http

import (
	"sync"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// Tests in this file cover the parts of server_mcp.go that do not require a
// real Repo: the static getMCPResources() catalog and the URI validation /
// dispatch layer in readMCPResource(). The handlers for /types, /objects, and
// /blobs each touch server.Repo and are intentionally left to BATS coverage
// (issue #150) where a real repo is wired up.

func TestGetMCPResourcesShape(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}
	resources := server.getMCPResources()

	if len(resources) == 0 {
		t.Fatalf("expected at least one MCP resource, got: 0")
	}

	uris := make(map[string]int, len(resources))
	for _, r := range resources {
		uris[r.URI]++
	}

	for _, expected := range []string{
		"dodder:///types",
		"dodder:///word-index",
	} {
		if uris[expected] != 1 {
			t.Fatalf("expected resource URI %q present exactly once, got count: %d (full list: %v)", expected, uris[expected], uris)
		}
	}

	for _, r := range resources {
		if r.Name == "" {
			t.Fatalf("resource %q has empty Name", r.URI)
		}
		if r.Description == "" {
			t.Fatalf("resource %q has empty Description", r.URI)
		}
		if r.MimeType != "application/json" {
			t.Fatalf("resource %q has MimeType %q, want application/json", r.URI, r.MimeType)
		}
	}
}

func TestReadMCPResourceMalformedURI(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}

	cases := []string{
		"",
		"not a uri at all",
	}

	for _, in := range cases {
		_, err := server.readMCPResource(in)
		if err == nil {
			t.Fatalf("readMCPResource(%q): expected error, got nil", in)
		}
	}
}

func TestReadMCPResourceWrongScheme(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}

	_, err := server.readMCPResource("http://example.com/types")
	if err == nil {
		t.Fatalf("expected error for non-dodder scheme, got nil")
	}
	if !errors.Is400BadRequest(err) {
		t.Fatalf("expected 400 Bad Request for wrong scheme, got: %v", err)
	}
}

func TestReadMCPResourceNonEmptyHostRejected(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}

	_, err := server.readMCPResource("dodder://somehost/types")
	if err == nil {
		t.Fatalf("expected error for non-empty host, got nil")
	}
	if !errors.Is400BadRequest(err) {
		t.Fatalf("expected 400 Bad Request for non-empty host, got: %v", err)
	}
}

func TestReadMCPResourceUnknownPath(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}

	cases := []string{
		"dodder:///does-not-exist",
		"dodder:///word-index",
		"dodder:///",
	}

	for _, in := range cases {
		_, err := server.readMCPResource(in)
		if err == nil {
			t.Fatalf("readMCPResource(%q): expected error for unknown path, got nil", in)
		}
		if !errors.Is400BadRequest(err) {
			t.Fatalf("readMCPResource(%q): expected 400 Bad Request, got: %v", in, err)
		}
	}
}

func TestReadMCPResourceConcurrentValidationDoesNotRace(t1 *testing.T) {
	t := ui.T{T: t1}

	server := &Server{}

	const N = 32

	inputs := []string{
		"dodder:///does-not-exist",
		"http://example.com/types",
		"dodder://withhost/types",
		"",
		"not a uri",
	}

	var wg sync.WaitGroup
	wg.Add(N)
	start := make(chan struct{})

	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			in := inputs[i%len(inputs)]
			_, err := server.readMCPResource(in)
			if err == nil {
				t1.Errorf("goroutine %d: input %q expected error, got nil", i, in)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	_ = t
}

//go:build test

package remote_http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
	"github.com/gorilla/mux"

	// Side-effect: register format-purpose pairs (matches production
	// binaries which pull this in via commands_dodder/main.go's blank
	// import). Required because the testing Repo's keypair generation
	// goes through piggy's markl format registry.
	_ "code.linenisgreat.com/dodder/go/internal/0/markl_registrations"
)

// TestHandleGetObjectMissingReturns404 pins the missing-OID semantics
// of /objects/{object_id} so edgeExplorer's IsErrNotFound branch keeps
// working. The hit path is covered end-to-end by clone_port.bats.
//
// Bypasses sigMiddleware (covered by sig_roundtrip_test.go) by
// registering only the route on a fresh mux.
func TestHandleGetObjectMissingReturns404(t1 *testing.T) {
	ui.RunTestContext(t1, testHandleGetObjectMissingReturns404)
}

func testHandleGetObjectMissingReturns404(t *ui.TestContext) {
	repo := local_working_copy.MakeTesting(t, nil)

	server := &Server{
		EnvLocal: repo,
		Repo:     repo,
	}

	t.AssertNoError(server.init())

	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc(
		"/objects/{object_id}",
		server.makeHandler(server.handleGetObject),
	).Methods("GET")

	// Type that doesn't exist in the freshly-initialized store —
	// edgeExplorer's first edge class is types, so missing types
	// are the realistic not-found shape.
	missingOid := "!nonexistent-type-v1"

	req := httptest.NewRequest(
		http.MethodGet,
		"/objects/"+url.QueryEscape(missingOid),
		nil,
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected 404 for missing OID %q, got %d (body: %s)",
			missingOid, rec.Code, rec.Body.String(),
		)
	}
}

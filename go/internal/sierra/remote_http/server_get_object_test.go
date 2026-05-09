//go:build test

package remote_http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/gorilla/mux"

	// Side-effect: register format-purpose pairs (matches production
	// binaries which pull this in via commands_dodder/main.go's blank
	// import). Required because the testing Repo's keypair generation
	// goes through madder's markl format registry.
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
)

// TestHandleGetObjectMissingReturns404 exercises the 404 branch of
// the new /objects/{object_id} route added for #171. The hit path is
// covered end-to-end by clone_port.bats once the URL transport is
// wired to httpRemoteObjectStore — that integration test is the
// real verification that the encoded body decodes correctly on the
// client side. This unit test pins the missing-OID semantics so
// edgeExplorer's IsErrNotFound branch keeps working.
//
// Bypasses sigMiddleware: that path is covered by sig_roundtrip_test.go.
// We register only the /objects route on a fresh mux so the test
// stays focused on handler behavior.
func TestHandleGetObjectMissingReturns404(t1 *testing.T) {
	ui.RunTestContext(t1, testHandleGetObjectMissingReturns404)
}

func testHandleGetObjectMissingReturns404(t *ui.TestContext) {
	repo := local_working_copy.MakeTesting(t, nil)

	server := &Server{
		EnvLocal: repo,
		Repo:     repo,
	}

	if err := server.init(); err != nil {
		t.Fatalf("server.init: %v", err)
	}

	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc(
		"/objects/{object_id}",
		server.makeHandler(server.handleGetObject),
	).Methods("GET")

	// A valid type OID that doesn't exist in the freshly-initialized
	// store. Picked because edgeExplorer's first edge-class is types,
	// so a type that's referenced-but-missing is the realistic
	// not-found shape.
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

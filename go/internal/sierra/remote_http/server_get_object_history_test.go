//go:build test

package remote_http

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
	"github.com/gorilla/mux"

	// Side-effect: register format-purpose pairs (see server_get_object_test).
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
)

// TestHandleGetObjectHistoryRoundTrip is a wiring smoke test for the
// /object-history route the parent negotiator relies on for merge resolution
// over the HTTP/stdio transport (#299). It pins that the route is registered,
// the handler runs server.Repo.ReadObjectHistory without panicking, the result
// serializes as an inventory list, and that blob decodes through the same codec
// the client uses (ReadInventoryListBlob, as in MakeInventoryList) — i.e. the
// full server->wire->client pipeline holds together.
//
// A fresh MakeTesting repo has an empty object stream (genesis types live in
// config, not as queryable stream objects), so the decoded list is empty here;
// that is fine for a wiring test. The content semantics — ReadObjectHistory
// returning every version, and the negotiator selecting the right base — are
// covered end-to-end by the direct-transport push.bats/pull.bats cases, which
// exercise the same ReadObjectHistory and the same negotiator.
//
// Bypasses sigMiddleware by registering only the route on a fresh mux (auth is
// covered by sig_roundtrip_test.go).
func TestHandleGetObjectHistoryRoundTrip(t1 *testing.T) {
	ui.RunTestContext(t1, testHandleGetObjectHistoryRoundTrip)
}

func testHandleGetObjectHistoryRoundTrip(t *ui.TestContext) {
	repo := local_working_copy.MakeTesting(t, nil)

	server := &Server{
		EnvLocal: repo,
		Repo:     repo,
	}

	t.AssertNoError(server.init())

	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc(
		"/object-history/{object_id}",
		server.makeHandler(server.handleGetObjectHistory),
	).Methods("GET")

	oid := "!md"

	req := httptest.NewRequest(
		http.MethodGet,
		"/object-history/"+url.QueryEscape(oid),
		nil,
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected 200 for %q, got %d (body: %s)",
			oid, rec.Code, rec.Body.String(),
		)
	}

	// The body must decode through the inventory-list codec the client uses;
	// an empty list is acceptable for a fresh repo.
	listTypeString := repo.GetImmutableConfigPublic().GetInventoryListTypeId()

	_, err := repo.GetInventoryListCoderCloset().ReadInventoryListBlob(
		repo.GetEnvRepo(),
		ids.GetOrPanic(listTypeString).TypeStruct,
		bufio.NewReader(rec.Body),
	)
	t.AssertNoError(err)
}

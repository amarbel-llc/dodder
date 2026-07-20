//go:build test

package remote_http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/sku_json_fmt"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
	"github.com/gorilla/mux"

	// Side-effect: register format-purpose pairs (matches production
	// binaries). Required because the testing Repo's keypair
	// generation goes through piggy's markl format registry.
	_ "code.linenisgreat.com/dodder/go/internal/0/markl_registrations"
)

// TestHandleGetQueryAcceptsJSON pins the content-negotiation contract
// the dodder-backed website API depends on: a query request carrying
// `Accept: application/json` is answered with a JSON array of
// sku_json_fmt.Transacted rather than the binary inventory-list wire
// format. Asserted against a freshly-initialized (empty-of-zettels)
// store, so the body is a well-formed empty array — proof the JSON
// branch is wired without depending on seed data.
func TestHandleGetQueryAcceptsJSON(t1 *testing.T) {
	ui.RunTestContext(t1, testHandleGetQueryAcceptsJSON)
}

func testHandleGetQueryAcceptsJSON(t *ui.TestContext) {
	repo := local_working_copy.MakeTesting(t, nil)

	server := &Server{
		EnvLocal: repo,
		Repo:     repo,
	}

	t.AssertNoError(server.init())

	router := mux.NewRouter().UseEncodedPath()
	router.HandleFunc(
		"/query/{list_type}/{query}",
		server.makeHandler(server.handleGetQuery),
	).Methods("GET")

	// list_type is intentionally arbitrary: the JSON branch never
	// touches the inventory-list codec, so an unconfigured type id
	// must not break negotiation.
	req := httptest.NewRequest(
		http.MethodGet,
		"/query/"+url.QueryEscape("inventory_list")+"/"+url.QueryEscape(":z"),
		nil,
	)
	req.Header.Set("Accept", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected 200, got %d (body: %s)",
			rec.Code, rec.Body.String(),
		)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected json content-type, got %q", ct)
	}

	var items []sku_json_fmt.Transacted

	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf(
			"response body is not a JSON array of objects: %s (body: %s)",
			err, rec.Body.String(),
		)
	}
}

// TestAcceptsJSON exercises the Accept-header negotiation predicate in
// isolation so its behavior can't drift silently.
func TestAcceptsJSON(t *testing.T) {
	cases := map[string]bool{
		"application/json":            true,
		"application/json; q=0.9":     true,
		"text/html, application/json": true,
		"text/html":                   false,
		"":                            false,
	}

	for accept, want := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}

		request := Request{Headers: req.Header}

		if got := acceptsJSON(request); got != want {
			t.Errorf("acceptsJSON(%q) = %v, want %v", accept, got, want)
		}
	}
}

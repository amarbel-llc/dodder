//go:build test

package remote_http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
	"github.com/gorilla/mux"
)

// TestObjectIdRoundTripsThroughURLPath verifies that #171's planned
// /objects/{object_id} endpoint can recover the OID strings that
// edgeExplorer actually produces (custom types, tags, zettel ids,
// referenced object ids). The wire pattern mirrors the existing
// /query/{list_type}/{query} route: client url.QueryEscape on send,
// server gorilla/mux UseEncodedPath + url.QueryUnescape on receive.
//
// If this test fails, /objects/{oid} can't be implemented as planned —
// the OID encoding scheme would need a different shape.
func TestObjectIdRoundTripsThroughURLPath(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Cover each shape edgeExplorer can produce, plus a few defensive
	// edge cases.
	cases := []struct {
		name string
		oid  string
	}{
		{"type", "!toml-type-v2"},
		{"type-with-version", "!md-v2"},
		{"tag-simple", "tag-name"},
		{"tag-dependent", "-dependent-tag"},
		{"tag-virtual", "%virtual-tag"},
		{"zettel", "one/uno"},
		{"zettel-with-suffix", "prefix/id-suffix"},
	}

	router := mux.NewRouter().UseEncodedPath()

	var (
		receivedString string
		receivedErr    error
	)

	router.HandleFunc(
		"/objects/{object_id}",
		func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			decoded, err := url.QueryUnescape(vars["object_id"])
			if err != nil {
				receivedErr = err
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			receivedString = decoded
			w.WriteHeader(http.StatusOK)
		},
	).Methods("GET")

	for _, tc := range cases {
		receivedString = ""
		receivedErr = nil

		var sent ids.ObjectId
		if err := sent.Set(tc.oid); err != nil {
			t.Errorf("[%s] Set(%q): %v", tc.name, tc.oid, err)
			continue
		}

		urlPath := "/objects/" + url.QueryEscape(sent.String())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", urlPath, nil)
		router.ServeHTTP(rec, req)

		if receivedErr != nil {
			t.Errorf(
				"[%s] server-side QueryUnescape: %v",
				tc.name, receivedErr,
			)
			continue
		}

		if rec.Code != http.StatusOK {
			t.Errorf(
				"[%s] status %d for %q (path %q)",
				tc.name, rec.Code, tc.oid, urlPath,
			)
			continue
		}

		var received ids.ObjectId
		if err := received.Set(receivedString); err != nil {
			t.Errorf(
				"[%s] server-side Set(%q): %v (sent=%q, urlPath=%q)",
				tc.name, receivedString, err, tc.oid, urlPath,
			)
			continue
		}

		if !sent.Equals(*received.GetObjectId()) {
			t.Errorf(
				"[%s] round-trip mismatch: sent=%q (genre=%v), received=%q (genre=%v), urlPath=%q",
				tc.name,
				sent.String(), sent.GetGenre(),
				received.String(), received.GetGenre(),
				urlPath,
			)
		}
	}
}

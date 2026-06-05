package remote_http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSigMiddlewarePublicModeGatesByMethod pins the public-mode bypass
// to safe (read-only) methods: with Server.Public set, a nonce-less GET
// or HEAD is served without attestation, but a nonce-less POST still
// falls through to the 400 sig-auth path so the mutating handlers
// (/blobs, /mcp, /inventory_lists) can't be reached unauthenticated.
// With Public unset, even a GET requires a nonce.
//
// Drives sigMiddleware directly so no Repo or keypair is needed: the
// bypass calls next without touching keys, and the 400 path returns at
// addSignatureIfNecessary's empty-nonce guard before any key use.
func TestSigMiddlewarePublicModeGatesByMethod(t *testing.T) {
	cases := []struct {
		name   string
		public bool
		method string
		want   int
	}{
		{"public GET bypasses", true, http.MethodGet, http.StatusOK},
		{"public HEAD bypasses", true, http.MethodHead, http.StatusOK},
		{"public POST blocked", true, http.MethodPost, http.StatusBadRequest},
		{"non-public GET blocked", false, http.MethodGet, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{Public: tc.public}

			handler := server.sigMiddleware(
				http.HandlerFunc(
					func(responseWriter http.ResponseWriter, _ *http.Request) {
						responseWriter.WriteHeader(http.StatusOK)
					},
				),
			)

			req := httptest.NewRequest(tc.method, "/objects/whatever", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf(
					"%s (public=%v): got %d, want %d (body: %s)",
					tc.method, tc.public,
					rec.Code, tc.want, rec.Body.String(),
				)
			}
		})
	}
}

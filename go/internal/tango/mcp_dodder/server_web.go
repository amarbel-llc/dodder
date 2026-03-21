package mcp_dodder

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type WebServer struct {
	Reader     ResourceReader
	CorsOrigin string
}

func (s *WebServer) Handler() http.Handler {
	r := mux.NewRouter()

	r.PathPrefix("/").HandlerFunc(s.handleResource).Methods("GET")
	r.PathPrefix("/").HandlerFunc(s.handleOptions).Methods("OPTIONS")

	return r
}

func (s *WebServer) handleResource(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	uri := "dodder://" + path

	result, err := s.Reader.ReadResource(r.Context(), uri)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			s.writeError(w, http.StatusNotFound, uri)
			return
		}
		s.writeError(w, http.StatusInternalServerError, errMsg)
		return
	}

	if len(result.Contents) == 0 {
		s.writeError(w, http.StatusNotFound, uri)
		return
	}

	content := result.Contents[0]
	mimeType := content.MimeType
	if mimeType == "" {
		mimeType = "application/json"
	}

	s.setCorsHeaders(w)
	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, content.Text)
}

func (s *WebServer) handleOptions(w http.ResponseWriter, r *http.Request) {
	s.setCorsHeaders(w)
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func (s *WebServer) setCorsHeaders(w http.ResponseWriter) {
	origin := s.CorsOrigin
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
}

func (s *WebServer) writeError(w http.ResponseWriter, status int, msg string) {
	s.setCorsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := struct {
		Error string `json:"error"`
	}{Error: msg}

	json.NewEncoder(w).Encode(resp)
}

func (s *WebServer) ListenAndServe(network, address string) error {
	listener, err := net.Listen(network, address)
	if err != nil {
		return fmt.Errorf("listen %s %s: %w", network, address, err)
	}

	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	fmt.Fprintf(os.Stderr,
		"starting HTTP server on port: %q\n",
		strconv.Itoa(addr.Port),
	)

	return http.Serve(listener, s.Handler())
}

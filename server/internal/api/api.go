// Package api wires the HTTP routes of the BI server.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/samsmon/hst-bi-poc/server/internal/sqlguard"
)

const maxBodyBytes = 256 * 1024

// New returns the router. fallback serves everything that is not an API
// route (the embedded web app); it may be nil.
func New(allowedSchemas []string, fallback http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/sql/validate", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid request body"})
			return
		}
		sql, err := sqlguard.Validate(body.Text, allowedSchemas)
		if err != nil {
			var ge *sqlguard.Error
			if errors.As(err, &ge) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": ge.Reason})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"sql": sql})
	})
	if fallback != nil {
		mux.Handle("/", fallback)
	}
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

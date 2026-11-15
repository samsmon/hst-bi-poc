// Package api wires the HTTP routes of the BI server.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/samsmon/hst-bi-poc/server/internal/db"
	"github.com/samsmon/hst-bi-poc/server/internal/semantic"
	"github.com/samsmon/hst-bi-poc/server/internal/sqlguard"
)

const maxBodyBytes = 256 * 1024

// Querier runs a read-only query. *db.DB implements it.
type Querier interface {
	Query(ctx context.Context, sql string, args []any, maxRows int) (*db.Result, error)
}

// Config is everything the router needs. DB may be nil: the data routes then
// answer 503 and the rest keeps working.
type Config struct {
	AllowedSchemas []string
	Models         map[string]*semantic.Model
	DB             Querier
	Fallback       http.Handler // serves the embedded web app; may be nil
}

// New returns the router.
func New(c Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/meta", c.meta)
	mux.HandleFunc("POST /api/sql/validate", c.validate)
	mux.HandleFunc("POST /api/query", c.query)
	mux.HandleFunc("POST /api/sql", c.sql)
	mux.HandleFunc("GET /api/schema", c.schema)
	if c.Fallback != nil {
		mux.Handle("/", c.Fallback)
	}
	return mux
}

func (c Config) meta(w http.ResponseWriter, _ *http.Request) {
	models := []semantic.Description{}
	for _, m := range c.Models {
		models = append(models, m.Describe())
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models, "database": c.DB != nil, "schemas": c.AllowedSchemas})
}

func (c Config) validate(w http.ResponseWriter, r *http.Request) {
	text, ok := readText(w, r)
	if !ok {
		return
	}
	sql, err := sqlguard.Validate(text, c.AllowedSchemas)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"sql": sql})
}

func readText(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, &body) {
		return "", false
	}
	return body.Text, true
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "invalid request body"})
		return false
	}
	return true
}

// writeError maps known error kinds to a status and a message that is safe
// to show. Anything else becomes a plain 500 without details.
func writeError(w http.ResponseWriter, err error) {
	var guard *sqlguard.Error
	var req *semantic.RequestError
	var qe *db.QueryError
	switch {
	case errors.As(err, &guard):
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": guard.Reason, "rejected": true})
	case errors.As(err, &req):
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": req.Reason})
	case errors.As(err, &qe) && qe.Timeout(), errors.Is(err, context.DeadlineExceeded):
		writeJSON(w, http.StatusGatewayTimeout, map[string]any{"detail": "The query was cancelled because it ran past the time limit.", "timeout": true})
	case errors.As(err, &qe):
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": qe.Message, "code": qe.Code})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

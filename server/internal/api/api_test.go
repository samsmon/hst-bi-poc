package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	h := New([]string{"public", "ml"}, nil)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(t, http.MethodGet, "/health", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateAcceptsSelect(t *testing.T) {
	rec := do(t, http.MethodPost, "/api/sql/validate", `{"text":"select 1;"}`)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"sql":"SELECT 1"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateRejectsWrite(t *testing.T) {
	rec := do(t, http.MethodPost, "/api/sql/validate", `{"text":"delete from t"}`)
	if rec.Code != 400 {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidateRejectsBadBodyAndMethod(t *testing.T) {
	if rec := do(t, http.MethodPost, "/api/sql/validate", `not json`); rec.Code != 400 {
		t.Fatalf("bad body: got %d", rec.Code)
	}
	if rec := do(t, http.MethodGet, "/api/sql/validate", ``); rec.Code != 405 {
		t.Fatalf("wrong method: got %d", rec.Code)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	big := `{"text":"` + strings.Repeat("a", 300_000) + `"}`
	if rec := do(t, http.MethodPost, "/api/sql/validate", big); rec.Code != 400 && rec.Code != 413 {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestFallbackHandlerServesUnknownPaths(t *testing.T) {
	h := New([]string{"public"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("spa"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboards/overview", nil))
	if rec.Body.String() != "spa" {
		t.Fatalf("got %q", rec.Body.String())
	}
}

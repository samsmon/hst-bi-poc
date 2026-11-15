package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samsmon/hst-bi-poc/server/internal/db"
	"github.com/samsmon/hst-bi-poc/server/internal/semantic"
)

// fake records what the handlers send to the database.
type fake struct {
	sql     string
	args    []any
	maxRows int
	res     *db.Result
	err     error
}

func (f *fake) Query(_ context.Context, sql string, args []any, maxRows int) (*db.Result, error) {
	f.sql, f.args, f.maxRows = sql, args, maxRows
	if f.err != nil {
		return nil, f.err
	}
	if f.res != nil {
		return f.res, nil
	}
	return &db.Result{Columns: []string{"a"}, Rows: [][]any{{1}}}, nil
}

func server(t *testing.T, q Querier) http.Handler {
	t.Helper()
	models, err := semantic.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	c := Config{AllowedSchemas: []string{"public", "bi", "ml"}, Models: models}
	if q != nil {
		c.DB = q
	}
	return New(c)
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(server(t, nil), http.MethodGet, "/health", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidate(t *testing.T) {
	h := server(t, nil)
	if rec := do(h, http.MethodPost, "/api/sql/validate", `{"text":"select 1;"}`); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"sql":"SELECT 1"`) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(h, http.MethodPost, "/api/sql/validate", `{"text":"delete from t"}`); rec.Code != 400 {
		t.Fatalf("got %d", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/api/sql/validate", `not json`); rec.Code != 400 {
		t.Fatalf("bad body: got %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/api/sql/validate", ``); rec.Code != 405 {
		t.Fatalf("wrong method: got %d", rec.Code)
	}
	big := `{"text":"` + strings.Repeat("a", 300_000) + `"}`
	if rec := do(h, http.MethodPost, "/api/sql/validate", big); rec.Code != 400 && rec.Code != 413 {
		t.Fatalf("oversized: got %d", rec.Code)
	}
}

func TestMetaListsModelWithoutSQL(t *testing.T) {
	rec := do(server(t, nil), http.MethodGet, "/api/meta", "")
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `"name":"monthly_sales"`) || !strings.Contains(body, `"database":false`) {
		t.Fatalf("got %d %s", rec.Code, body)
	}
	if strings.Contains(body, "mv_monthly_sales") || strings.Contains(body, "sum(") {
		t.Fatalf("meta leaks SQL: %s", body)
	}
}

func TestDataRoutesNeedADatabase(t *testing.T) {
	h := server(t, nil)
	for _, r := range [][3]string{{"POST", "/api/query", `{"measures":["tickets"]}`}, {"POST", "/api/sql", `{"text":"select 1"}`}, {"GET", "/api/schema", ""}} {
		if rec := do(h, r[0], r[1], r[2]); rec.Code != 503 {
			t.Errorf("%s %s: got %d", r[0], r[1], rec.Code)
		}
	}
}

func TestQueryCompilesAndRuns(t *testing.T) {
	f := &fake{}
	rec := do(server(t, f), http.MethodPost, "/api/query", `{"measures":["tickets"],"dimensions":["channel"],"filters":[{"dimension":"season_type","in":["normal"]}]}`)
	if rec.Code != 200 {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(f.sql, "FROM bi.mv_monthly_sales") || len(f.args) != 1 || f.maxRows != 5000 {
		t.Fatalf("sql %q args %v max %d", f.sql, f.args, f.maxRows)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out["columns"] == nil || out["sql"] == nil {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestQueryRejectsUnknownNamesAndModels(t *testing.T) {
	f := &fake{}
	h := server(t, f)
	if rec := do(h, http.MethodPost, "/api/query", `{"measures":["nope"]}`); rec.Code != 400 {
		t.Fatalf("got %d", rec.Code)
	}
	if rec := do(h, http.MethodPost, "/api/query", `{"model":"other","measures":["tickets"]}`); rec.Code != 400 {
		t.Fatalf("got %d", rec.Code)
	}
	if f.sql != "" {
		t.Fatalf("database was reached: %s", f.sql)
	}
}

func TestSQLRunsOnlyWhatTheGuardReturned(t *testing.T) {
	f := &fake{}
	rec := do(server(t, f), http.MethodPost, "/api/sql", `{"text":"select season_type from public.fact_ticket_sales_wide; "}`)
	if rec.Code != 200 {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	if f.sql != "SELECT season_type FROM public.fact_ticket_sales_wide" || f.maxRows != 10000 {
		t.Fatalf("sql %q max %d", f.sql, f.maxRows)
	}
}

func TestSQLRejectionNeverReachesTheDatabase(t *testing.T) {
	f := &fake{}
	rec := do(server(t, f), http.MethodPost, "/api/sql", `{"text":"delete from public.fact_ticket_sales_wide"}`)
	if rec.Code != 400 || f.sql != "" {
		t.Fatalf("got %d, sql %q", rec.Code, f.sql)
	}
	if !strings.Contains(rec.Body.String(), `"rejected":true`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestDatabaseErrorsAreMapped(t *testing.T) {
	h := server(t, &fake{err: &db.QueryError{Message: `column "x" does not exist`, Code: "42703"}})
	rec := do(h, http.MethodPost, "/api/sql", `{"text":"select x from t"}`)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "does not exist") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	h = server(t, &fake{err: &db.QueryError{Message: "canceling statement due to statement timeout", Code: "57014"}})
	if rec := do(h, http.MethodPost, "/api/sql", `{"text":"select 1"}`); rec.Code != 504 {
		t.Fatalf("timeout: got %d", rec.Code)
	}
	h = server(t, &fake{err: context.DeadlineExceeded})
	if rec := do(h, http.MethodPost, "/api/query", `{"measures":["tickets"]}`); rec.Code != 504 {
		t.Fatalf("deadline: got %d", rec.Code)
	}
}

func TestSchemaGroupsColumnsByTable(t *testing.T) {
	f := &fake{res: &db.Result{Columns: []string{"s", "t", "k", "n", "c", "ty", "d"}, Rows: [][]any{
		{"public", "fact", "r", int64(10), "id", "bigint", ""},
		{"public", "fact", "r", int64(10), "name", "text", "Customer name"},
		{"bi", "mv", "m", int64(5), "month", "date", ""},
	}}}
	rec := do(server(t, f), http.MethodGet, "/api/schema", "")
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `"name":"fact"`) || !strings.Contains(body, `"description":"Customer name"`) || !strings.Contains(body, `"kind":"materialized view"`) {
		t.Fatalf("got %d %s", rec.Code, body)
	}
	if len(f.args) != 1 {
		t.Fatalf("schema list not passed as a parameter: %v", f.args)
	}
}

func TestFallbackServesUnknownPaths(t *testing.T) {
	h := New(Config{Fallback: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("spa")) })})
	if rec := do(h, http.MethodGet, "/dashboards/overview", ""); rec.Body.String() != "spa" {
		t.Fatalf("got %q", rec.Body.String())
	}
}

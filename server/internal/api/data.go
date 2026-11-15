package api

import (
	"context"
	"net/http"
	"time"

	"github.com/samsmon/hst-bi-poc/server/internal/db"
	"github.com/samsmon/hst-bi-poc/server/internal/semantic"
	"github.com/samsmon/hst-bi-poc/server/internal/sqlguard"
)

const (
	defaultModel    = "monthly_sales"
	dashboardRows   = 5000
	consoleRows     = 10000
	dashboardBudget = 15 * time.Second
	consoleBudget   = 35 * time.Second // a little above the role's 30 s timeout
)

// schemaSQL lists readable tables, views and materialized views with their
// columns. information_schema leaves materialized views out, so it reads the
// catalog; the schema list is a bound parameter.
const schemaSQL = `SELECT n.nspname, c.relname, c.relkind::text, greatest(c.reltuples, 0)::bigint,
       a.attname, format_type(a.atttypid, a.atttypmod), coalesce(col_description(c.oid, a.attnum), '')
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid
WHERE n.nspname = ANY($1) AND c.relkind IN ('r', 'v', 'm') AND a.attnum > 0 AND NOT a.attisdropped
  AND has_table_privilege(c.oid, 'SELECT')
ORDER BY n.nspname, c.relname, a.attnum`

type queryResponse struct {
	*db.Result
	SQL string `json:"sql"`
}

func (c Config) ready(w http.ResponseWriter) bool {
	if c.DB == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": "The database is not configured on this server."})
		return false
	}
	return true
}

func (c Config) query(w http.ResponseWriter, r *http.Request) {
	if !c.ready(w) {
		return
	}
	var req semantic.Request
	if !readJSON(w, r, &req) {
		return
	}
	if req.Model == "" {
		req.Model = defaultModel
	}
	model, ok := c.Models[req.Model]
	if !ok {
		writeError(w, &semantic.RequestError{Reason: "unknown model " + req.Model})
		return
	}
	q, err := model.Compile(req)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dashboardBudget)
	defer cancel()
	res, err := c.DB.Query(ctx, q.SQL, q.Args, dashboardRows)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, queryResponse{Result: res, SQL: q.SQL})
}

func (c Config) sql(w http.ResponseWriter, r *http.Request) {
	if !c.ready(w) {
		return
	}
	text, ok := readText(w, r)
	if !ok {
		return
	}
	safe, err := sqlguard.Validate(text, c.AllowedSchemas)
	if err != nil {
		writeError(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), consoleBudget)
	defer cancel()
	res, err := c.DB.Query(ctx, safe, nil, consoleRows)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, queryResponse{Result: res, SQL: safe})
}

type column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type table struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Rows    int64    `json:"rows"`
	Columns []column `json:"columns"`
}

var kinds = map[string]string{"r": "table", "v": "view", "m": "materialized view"}

func (c Config) schema(w http.ResponseWriter, r *http.Request) {
	if !c.ready(w) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dashboardBudget)
	defer cancel()
	res, err := c.DB.Query(ctx, schemaSQL, []any{c.AllowedSchemas}, 20000)
	if err != nil {
		writeError(w, err)
		return
	}
	tables := []*table{}
	for _, row := range res.Rows {
		if len(row) != 7 {
			continue
		}
		s, _ := row[0].(string)
		n, _ := row[1].(string)
		k, _ := row[2].(string)
		rows, _ := row[3].(int64)
		last := len(tables) - 1
		if last < 0 || tables[last].Schema != s || tables[last].Name != n {
			tables = append(tables, &table{Schema: s, Name: n, Kind: kinds[k], Rows: rows})
			last++
		}
		name, _ := row[4].(string)
		typ, _ := row[5].(string)
		desc, _ := row[6].(string)
		tables[last].Columns = append(tables[last].Columns, column{Name: name, Type: typ, Description: desc})
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

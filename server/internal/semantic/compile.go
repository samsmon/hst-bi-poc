package semantic

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultLimit = 1000
	maxLimit     = 5000
)

// Request is what the UI sends: names only, plus filter values.
type Request struct {
	Model      string   `json:"model"`
	Measures   []string `json:"measures"`
	Dimensions []string `json:"dimensions"`
	Grain      string   `json:"grain"`
	Filters    []Filter `json:"filters"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Order      []Order  `json:"order"`
	Limit      int      `json:"limit"`
}

// Filter keeps rows whose dimension value is in the list.
type Filter struct {
	Dimension string   `json:"dimension"`
	In        []string `json:"in"`
}

// Order sorts by a selected column.
type Order struct {
	By  string `json:"by"`
	Dir string `json:"dir"`
}

// Query is ready to run: SQL with placeholders, its arguments, column names.
type Query struct {
	SQL     string
	Args    []any
	Columns []string
}

// RequestError marks a problem the caller can fix (HTTP 400).
type RequestError struct{ Reason string }

func (e *RequestError) Error() string { return e.Reason }

func bad(format string, args ...any) error {
	return &RequestError{Reason: fmt.Sprintf(format, args...)}
}

// Compile turns a request into SQL. Only names from the model file reach the
// SQL text; dates and filter values are bound as parameters.
func (m *Model) Compile(r Request) (Query, error) {
	var q Query
	var selects, where []string
	seen := map[string]bool{}
	add := func(expr, name string) error {
		if seen[name] {
			return bad("%s is selected twice", name)
		}
		seen[name] = true
		selects = append(selects, expr+" AS "+name)
		q.Columns = append(q.Columns, name)
		return nil
	}

	groupCols := 0
	switch r.Grain {
	case "":
	case "month":
		_ = add(m.TimeColumn, "period")
		groupCols++
	case "quarter", "year":
		_ = add(fmt.Sprintf("date_trunc('%s', %s)::date", r.Grain, m.TimeColumn), "period")
		groupCols++
	default:
		return q, bad("unknown grain %q", r.Grain)
	}
	for _, d := range r.Dimensions {
		f, ok := m.byDim[d]
		if !ok {
			return q, bad("unknown dimension %q", d)
		}
		if err := add(f.SQL, f.Name); err != nil {
			return q, err
		}
		groupCols++
	}
	if len(r.Measures) == 0 {
		return q, bad("at least one measure is required")
	}
	for _, name := range r.Measures {
		f, ok := m.byMeasure[name]
		if !ok {
			return q, bad("unknown measure %q", name)
		}
		if err := add(f.SQL, f.Name); err != nil {
			return q, err
		}
	}

	arg := func(v any) string {
		q.Args = append(q.Args, v)
		return fmt.Sprintf("$%d", len(q.Args))
	}
	for _, bound := range []struct{ val, op string }{{r.From, ">="}, {r.To, "<="}} {
		if bound.val == "" {
			continue
		}
		if _, err := time.Parse("2006-01-02", bound.val); err != nil {
			return q, bad("dates must look like 2025-01-31")
		}
		where = append(where, fmt.Sprintf("%s %s %s::date", m.TimeColumn, bound.op, arg(bound.val)))
	}
	for _, f := range r.Filters {
		d, ok := m.byDim[f.Dimension]
		if !ok {
			return q, bad("unknown filter dimension %q", f.Dimension)
		}
		if len(f.In) == 0 {
			return q, bad("filter on %s has no values", f.Dimension)
		}
		where = append(where, fmt.Sprintf("%s = ANY(%s)", wrap(d.SQL), arg(f.In)))
	}

	order, err := orderBy(r.Order, q.Columns, r.Grain != "", groupCols)
	if err != nil {
		return q, err
	}
	limit := r.Limit
	switch {
	case limit < 0 || limit > maxLimit:
		return q, bad("limit must be between 1 and %d", maxLimit)
	case limit == 0:
		limit = defaultLimit
	}

	var b strings.Builder
	b.WriteString("SELECT " + strings.Join(selects, ", ") + " FROM " + m.Source)
	if len(where) > 0 {
		b.WriteString(" WHERE " + strings.Join(where, " AND "))
	}
	if groupCols > 0 {
		nums := make([]string, groupCols)
		for i := range nums {
			nums[i] = fmt.Sprint(i + 1)
		}
		b.WriteString(" GROUP BY " + strings.Join(nums, ", "))
		b.WriteString(" ORDER BY " + order)
	}
	fmt.Fprintf(&b, " LIMIT %d", limit)
	q.SQL = b.String()
	return q, nil
}

// wrap parenthesises expressions so "a || b = ANY(...)" binds as intended.
func wrap(expr string) string {
	if strings.ContainsAny(expr, " |") {
		return "(" + expr + ")"
	}
	return expr
}

func orderBy(orders []Order, cols []string, hasGrain bool, groupCols int) (string, error) {
	if len(orders) == 0 {
		if hasGrain {
			return "1 ASC", nil
		}
		return fmt.Sprintf("%d DESC", groupCols+1), nil // first measure
	}
	var parts []string
	for _, o := range orders {
		pos := 0
		for i, c := range cols {
			if c == o.By {
				pos = i + 1
			}
		}
		if pos == 0 {
			return "", bad("cannot order by %q: it is not selected", o.By)
		}
		dir := strings.ToUpper(o.Dir)
		if dir == "" {
			dir = "ASC"
		}
		if dir != "ASC" && dir != "DESC" {
			return "", bad("order direction must be asc or desc")
		}
		parts = append(parts, fmt.Sprintf("%d %s", pos, dir))
	}
	return strings.Join(parts, ", "), nil
}

package semantic

import (
	"reflect"
	"strings"
	"testing"
)

func model(t *testing.T) *Model {
	t.Helper()
	models, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := models["monthly_sales"]
	if !ok {
		t.Fatal("monthly_sales model missing")
	}
	return m
}

func TestCompileTotals(t *testing.T) {
	q, err := model(t).Compile(Request{Measures: []string{"tickets", "net_revenue"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT sum(tickets) AS tickets, sum(net_revenue) AS net_revenue FROM bi.mv_monthly_sales LIMIT 1000"
	if q.SQL != want {
		t.Fatalf("got  %s\nwant %s", q.SQL, want)
	}
	if len(q.Args) != 0 || !reflect.DeepEqual(q.Columns, []string{"tickets", "net_revenue"}) {
		t.Fatalf("args %v columns %v", q.Args, q.Columns)
	}
}

func TestCompileGrainDimensionsFiltersAndRange(t *testing.T) {
	q, err := model(t).Compile(Request{
		Measures:   []string{"net_revenue"},
		Dimensions: []string{"season_type"},
		Grain:      "year",
		Filters:    []Filter{{Dimension: "channel", In: []string{"App", "Web"}}},
		From:       "2024-01-01",
		To:         "2025-12-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT date_trunc('year', month)::date AS period, season_type AS season_type, sum(net_revenue) AS net_revenue" +
		" FROM bi.mv_monthly_sales" +
		" WHERE month >= $1::date AND month <= $2::date AND sales_channel_name = ANY($3)" +
		" GROUP BY 1, 2 ORDER BY 1 ASC LIMIT 1000"
	if q.SQL != want {
		t.Fatalf("got  %s\nwant %s", q.SQL, want)
	}
	wantArgs := []any{"2024-01-01", "2025-12-31", []string{"App", "Web"}}
	if !reflect.DeepEqual(q.Args, wantArgs) {
		t.Fatalf("args %v", q.Args)
	}
	if !reflect.DeepEqual(q.Columns, []string{"period", "season_type", "net_revenue"}) {
		t.Fatalf("columns %v", q.Columns)
	}
}

func TestCompileOrderAndLimit(t *testing.T) {
	q, err := model(t).Compile(Request{
		Measures:   []string{"tickets"},
		Dimensions: []string{"od"},
		Order:      []Order{{By: "tickets", Dir: "desc"}},
		Limit:      5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(q.SQL, "GROUP BY 1 ORDER BY 2 DESC LIMIT 5") {
		t.Fatalf("got %s", q.SQL)
	}
	if !strings.Contains(q.SQL, "origin_station_name || ' - ' || destination_station_name AS od") {
		t.Fatalf("got %s", q.SQL)
	}
}

func TestCompileDefaultsOrderToFirstMeasureWithoutGrain(t *testing.T) {
	q, err := model(t).Compile(Request{Measures: []string{"tickets"}, Dimensions: []string{"channel"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(q.SQL, "GROUP BY 1 ORDER BY 2 DESC LIMIT 1000") {
		t.Fatalf("got %s", q.SQL)
	}
}

func TestCompileRejectsBadRequests(t *testing.T) {
	bad := map[string]Request{
		"no measures":        {},
		"unknown measure":    {Measures: []string{"nope"}},
		"unknown dimension":  {Measures: []string{"tickets"}, Dimensions: []string{"nope"}},
		"injection as name":  {Measures: []string{"tickets; drop table x"}},
		"unknown grain":      {Measures: []string{"tickets"}, Grain: "week"},
		"bad date":           {Measures: []string{"tickets"}, From: "2024-13-40"},
		"date with sql":      {Measures: []string{"tickets"}, To: "2024-01-01'; --"},
		"unknown filter":     {Measures: []string{"tickets"}, Filters: []Filter{{Dimension: "nope", In: []string{"a"}}}},
		"empty filter":       {Measures: []string{"tickets"}, Filters: []Filter{{Dimension: "channel"}}},
		"order not selected": {Measures: []string{"tickets"}, Order: []Order{{By: "net_revenue"}}},
		"bad direction":      {Measures: []string{"tickets"}, Order: []Order{{By: "tickets", Dir: "sideways"}}},
		"duplicate measure":  {Measures: []string{"tickets", "tickets"}},
		"limit too large":    {Measures: []string{"tickets"}, Limit: 1_000_000},
		"negative limit":     {Measures: []string{"tickets"}, Limit: -1},
	}
	for name, req := range bad {
		if q, err := model(t).Compile(req); err == nil {
			t.Errorf("%s: want error, got %s", name, q.SQL)
		}
	}
}

func TestFilterValuesNeverReachTheSQLText(t *testing.T) {
	evil := "x'); DROP TABLE t; --"
	q, err := model(t).Compile(Request{Measures: []string{"tickets"}, Filters: []Filter{{Dimension: "channel", In: []string{evil}}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q.SQL, "DROP") {
		t.Fatalf("value leaked into SQL: %s", q.SQL)
	}
}

func TestDescribeListsNamesInFileOrder(t *testing.T) {
	d := model(t).Describe()
	if d.Measures[0].Name != "tickets" || d.Dimensions[0].Name != "season_type" {
		t.Fatalf("order lost: %v %v", d.Measures[0], d.Dimensions[0])
	}
	if d.Measures[4].Format != "percent" {
		t.Fatalf("format lost: %+v", d.Measures[4])
	}
}

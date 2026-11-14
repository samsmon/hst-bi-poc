package sqlguard

import (
	"strings"
	"testing"
)

var allowed = []string{"public", "ml"}

func TestAcceptsReadOnlyQueries(t *testing.T) {
	cases := []string{
		"SELECT 1",
		"select count(*) from public.fact_ticket_sales_wide",
		"SELECT season_type, sum(net_revenue) FROM fact_ticket_sales_wide GROUP BY 1",
		"WITH a AS (SELECT 1 AS x) SELECT * FROM a",
		"SELECT 1 UNION ALL SELECT 2",
		"SELECT date_trunc('month', travel_date), count(*) FROM t GROUP BY 1",
		"SELECT sum(x) OVER (PARTITION BY y ORDER BY z) FROM t",
		"SELECT * FROM ml.forecast WHERE target_name = 'ticket_sales'",
		"SELECT 1;",
		"EXPLAIN SELECT 1",
		"SELECT * FROM generate_series(1, 5)",
		"SELECT 'a;b' AS x",
		"/* hi */ SeLeCt 1 -- trailing",
	}
	for _, sql := range cases {
		if _, err := Validate(sql, allowed); err != nil {
			t.Errorf("want accepted: %q, got %v", sql, err)
		}
	}
}

func TestRejects(t *testing.T) {
	groups := map[string][]string{
		"non-select statements": {
			"SELECT 1; SELECT 2",
			"INSERT INTO t VALUES (1)",
			"UPDATE t SET a = 1",
			"DELETE FROM t",
			"TRUNCATE t",
			"DROP TABLE t",
			"ALTER TABLE t ADD COLUMN c int",
			"CREATE TABLE t (a int)",
			"COPY t TO STDOUT",
			"SET statement_timeout = 0",
			"RESET ALL",
			"DO $$ BEGIN PERFORM 1; END $$",
			"CALL p()",
			"VACUUM t",
			"LISTEN c",
			"NOTIFY c",
			"GRANT ALL ON t TO PUBLIC",
			"BEGIN",
			"",
			"   ",
			"-- only a comment",
			"SELECT 1 /* ; */ ; DrOp TABLE t",
		},
		"denied functions": {
			"SELECT pg_sleep(100)",
			"SELECT pg_read_file('/etc/passwd')",
			"SELECT * FROM dblink('host=x', 'select 1') AS t(a int)",
			"SELECT lo_import('/etc/passwd')",
			"SELECT set_config('statement_timeout', '0', false)",
			"SELECT pg_terminate_backend(1)",
			"SELECT PG_SLEEP(1)",
			"SELECT pg_catalog.pg_sleep(1)",
			"SELECT nextval('s')",
			"SELECT setval('s', 1)",
			"SELECT * FROM t WHERE x IN (SELECT pg_sleep(1))",
			`SELECT "pg_sleep"(1)`,
			"SELECT * FROM t, LATERAL pg_sleep(1) s",
		},
		"schemas outside the allow list": {
			"SELECT * FROM pg_shadow",
			"SELECT * FROM pg_catalog.pg_shadow",
			"SELECT * FROM information_schema.tables",
			"SELECT * FROM other_app.secrets",
			"SELECT * FROM public.a JOIN other_app.b ON true",
			`SELECT * FROM "other_app"."secrets"`,
			"SELECT * FROM (SELECT * FROM other_app.t) q",
		},
		"select with side effects": {
			"SELECT * INTO new_table FROM t",
			"SELECT * FROM t FOR UPDATE",
			"SELECT * FROM t FOR SHARE",
			"SELECT 1 FROM t FOR NO KEY UPDATE",
			"WITH d AS (DELETE FROM t RETURNING *) SELECT * FROM d",
			"WITH i AS (INSERT INTO t VALUES (1) RETURNING *) SELECT * FROM i",
		},
		"explain that could execute": {
			"EXPLAIN ANALYZE SELECT 1",
			"EXPLAIN (ANALYZE) SELECT 1",
			"EXPLAIN INSERT INTO t VALUES (1)",
		},
	}
	for name, cases := range groups {
		for _, sql := range cases {
			if out, err := Validate(sql, allowed); err == nil {
				t.Errorf("%s: want rejected: %q, got %q", name, sql, out)
			}
		}
	}
}

func TestRejectsTooLongInput(t *testing.T) {
	sql := "SELECT " + strings.Repeat("1,", 100_000) + "1"
	if _, err := Validate(sql, allowed); err == nil {
		t.Fatal("want rejected for oversized input")
	}
}

func TestReturnsRegeneratedSQL(t *testing.T) {
	out, err := Validate("select 1;", allowed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.ToLower(out), "select 1") || strings.Contains(out, ";") {
		t.Fatalf("unexpected output %q", out)
	}
}

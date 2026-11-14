import pytest

from app.sqlguard import GuardError, validate

ALLOWED = {"public", "ml"}


def ok(sql: str) -> str:
    return validate(sql, ALLOWED)


@pytest.mark.parametrize(
    "sql",
    [
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
    ],
)
def test_accepts_read_only_queries(sql):
    assert ok(sql)


@pytest.mark.parametrize(
    "sql",
    [
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
    ],
)
def test_rejects_non_select_statements(sql):
    with pytest.raises(GuardError):
        ok(sql)


@pytest.mark.parametrize(
    "sql",
    [
        "SELECT pg_sleep(100)",
        "SELECT pg_read_file('/etc/passwd')",
        "SELECT * FROM dblink('host=x', 'select 1') AS t(a int)",
        "SELECT lo_import('/etc/passwd')",
        "SELECT set_config('statement_timeout', '0', false)",
        "SELECT pg_terminate_backend(1)",
        "SELECT PG_SLEEP(1)",
        "SELECT nextval('s')",
        "SELECT setval('s', 1)",
        "SELECT * FROM t WHERE x IN (SELECT pg_sleep(1))",
        'SELECT "pg_sleep"(1)',
        "SELECT pg_catalog.pg_sleep(1)",
    ],
)
def test_rejects_denied_functions(sql):
    with pytest.raises(GuardError):
        ok(sql)


@pytest.mark.parametrize(
    "sql",
    [
        "SELECT * FROM pg_shadow",
        "SELECT * FROM pg_catalog.pg_shadow",
        "SELECT * FROM information_schema.tables",
        "SELECT * FROM other_app.secrets",
        "SELECT * FROM public.a JOIN other_app.b ON true",
    ],
)
def test_rejects_schemas_outside_allow_list(sql):
    with pytest.raises(GuardError):
        ok(sql)


@pytest.mark.parametrize(
    "sql",
    [
        "SELECT * INTO new_table FROM t",
        "SELECT * FROM t FOR UPDATE",
        "SELECT * FROM t FOR SHARE",
        "WITH d AS (DELETE FROM t RETURNING *) SELECT * FROM d",
        "WITH i AS (INSERT INTO t VALUES (1) RETURNING *) SELECT * FROM i",
    ],
)
def test_rejects_select_with_side_effects(sql):
    with pytest.raises(GuardError):
        ok(sql)


@pytest.mark.parametrize(
    "sql",
    [
        "EXPLAIN ANALYZE SELECT 1",
        "EXPLAIN (ANALYZE) SELECT 1",
        "EXPLAIN INSERT INTO t VALUES (1)",
    ],
)
def test_explain_never_executes(sql):
    with pytest.raises(GuardError):
        ok(sql)


def test_comments_and_case_do_not_hide_statements():
    with pytest.raises(GuardError):
        ok("SELECT 1 /* ; */ ; DrOp TABLE t")
    assert ok("/* hi */ SeLeCt 1 -- trailing")


def test_string_literal_with_semicolon_is_fine():
    assert ok("SELECT 'a;b' AS x")


def test_quoted_identifier_cannot_smuggle_a_schema():
    with pytest.raises(GuardError):
        ok('SELECT * FROM "other_app"."secrets"')


def test_too_long_input_is_rejected():
    with pytest.raises(GuardError):
        ok("SELECT " + "1," * 100_000 + "1")


def test_returns_sql_text_that_runs_as_given_statement():
    out = ok("select 1;")
    assert out.lower().startswith("select 1")
    assert ";" not in out

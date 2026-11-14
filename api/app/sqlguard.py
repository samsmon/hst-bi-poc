"""Validate user SQL for the read-only console.

The database role is the real safety boundary. This guard gives early,
readable errors and returns SQL regenerated from the parsed tree, so the
database runs exactly what was validated (no parser differential).
"""

import re

import sqlglot
from sqlglot import exp
from sqlglot.errors import SqlglotError

MAX_SQL_CHARS = 100_000

DENIED_FUNCTION_PREFIXES = ("pg_", "lo_", "dblink", "txid_")
DENIED_FUNCTIONS = {"set_config", "nextval", "setval", "currval", "query_to_xml", "xpath"}
DENIED_SCHEMAS = {"pg_catalog", "information_schema"}

_SIDE_EFFECT_NODES = tuple(
    getattr(exp, name)
    for name in (
        "Insert", "Update", "Delete", "Merge", "Create", "Drop", "Alter",
        "Command", "Copy", "Into", "Lock", "Set", "Use", "TruncateTable",
        "Grant", "Transaction", "Commit", "Rollback",
    )
    if hasattr(exp, name)
)

_EXPLAIN = re.compile(r"^\s*explain\s+(?P<rest>.+)$", re.IGNORECASE | re.DOTALL)
_EXPLAIN_FLAGS = re.compile(r"^\s*(\(|analy[sz]e\b|verbose\b)", re.IGNORECASE)


class GuardError(ValueError):
    """The submitted SQL is not allowed."""


def validate(sql: str, allowed_schemas: set[str]) -> str:
    if len(sql) > MAX_SQL_CHARS:
        raise GuardError(f"SQL is longer than {MAX_SQL_CHARS} characters")

    explain = _EXPLAIN.match(sql)
    if explain:
        rest = explain.group("rest")
        if _EXPLAIN_FLAGS.match(rest):
            raise GuardError("EXPLAIN options and ANALYZE are not allowed")
        return "EXPLAIN " + validate(rest, allowed_schemas)

    try:
        statements = [s for s in sqlglot.parse(sql, dialect="postgres") if s is not None]
    except SqlglotError as err:
        raise GuardError(f"SQL could not be parsed: {err}") from err

    if len(statements) != 1:
        raise GuardError("Exactly one SELECT statement is required")
    root = statements[0]
    if not isinstance(root, exp.Query):
        raise GuardError("Only SELECT statements are allowed")

    _check_nodes(root)
    _check_functions(root)
    _check_tables(root, {s.lower() for s in allowed_schemas})
    return root.sql(dialect="postgres", comments=False)


def _check_nodes(root: exp.Expression) -> None:
    for node in root.walk():
        if isinstance(node, _SIDE_EFFECT_NODES):
            raise GuardError(f"{type(node).__name__} is not allowed in a read-only query")


def _check_functions(root: exp.Expression) -> None:
    for node in root.find_all(exp.Func):
        name = (node.name if isinstance(node, exp.Anonymous) else node.sql_name()).lower()
        if name in DENIED_FUNCTIONS or name.startswith(DENIED_FUNCTION_PREFIXES):
            raise GuardError(f"Function {name} is not allowed")


def _check_tables(root: exp.Expression, allowed: set[str]) -> None:
    ctes = {c.alias.lower() for c in root.find_all(exp.CTE)}
    for table in root.find_all(exp.Table):
        schema = table.db.lower()
        name = table.name.lower()
        if schema:
            if schema in DENIED_SCHEMAS or schema not in allowed:
                raise GuardError(f"Schema {schema} is not allowed")
        elif name not in ctes and (name.startswith("pg_") or name in DENIED_SCHEMAS):
            raise GuardError(f"Table {name} is not allowed")

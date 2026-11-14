// Package sqlguard validates user SQL for the read-only console.
//
// The database role is the real safety boundary. This guard gives early,
// readable errors and returns SQL deparsed from the parsed tree, so the
// database runs exactly what was validated.
package sqlguard

import (
	"encoding/json"
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// MaxSQLChars is the longest accepted input.
const MaxSQLChars = 100_000

var (
	deniedFunctionPrefixes = []string{"pg_", "lo_", "dblink", "txid_"}
	deniedFunctions        = map[string]bool{
		"set_config": true, "nextval": true, "setval": true, "currval": true,
		"query_to_xml": true, "xpath": true,
	}
	deniedSchemas = map[string]bool{"pg_catalog": true, "information_schema": true}
	// Node kinds that must never appear anywhere in the tree.
	sideEffectNodes = map[string]bool{
		"InsertStmt": true, "UpdateStmt": true, "DeleteStmt": true, "MergeStmt": true,
		"IntoClause": true, "LockingClause": true,
	}
)

// Error means the submitted SQL is not allowed.
type Error struct{ Reason string }

func (e *Error) Error() string { return e.Reason }

func reject(format string, args ...any) error {
	return &Error{Reason: fmt.Sprintf(format, args...)}
}

// Validate returns the deparsed SQL when sql is a single read-only SELECT
// (or a plain EXPLAIN of one) that only touches allowedSchemas.
func Validate(sql string, allowedSchemas []string) (string, error) {
	if len(sql) > MaxSQLChars {
		return "", reject("SQL is longer than %d characters", MaxSQLChars)
	}
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return "", reject("SQL could not be parsed: %v", err)
	}
	if len(tree.Stmts) != 1 {
		return "", reject("Exactly one SELECT statement is required")
	}
	raw, err := pg_query.ParseToJSON(sql)
	if err != nil {
		return "", reject("SQL could not be parsed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return "", reject("internal: cannot read parse tree")
	}
	stmts, _ := doc["stmts"].([]any)
	if len(stmts) != 1 {
		return "", reject("Exactly one SELECT statement is required")
	}
	root, _ := stmts[0].(map[string]any)["stmt"].(map[string]any)

	if err := checkRoot(root); err != nil {
		return "", err
	}

	allowed := make(map[string]bool, len(allowedSchemas))
	for _, s := range allowedSchemas {
		allowed[strings.ToLower(s)] = true
	}
	ctes := map[string]bool{}
	walk(doc, func(kind string, node map[string]any) {
		if kind == "CommonTableExpr" {
			if name, _ := node["ctename"].(string); name != "" {
				ctes[strings.ToLower(name)] = true
			}
		}
	})
	var firstErr error
	walk(doc, func(kind string, node map[string]any) {
		if firstErr != nil {
			return
		}
		switch {
		case sideEffectNodes[kind]:
			firstErr = reject("%s is not allowed in a read-only query", kind)
		case kind == "SelectStmt" && node["intoClause"] != nil:
			firstErr = reject("SELECT INTO is not allowed in a read-only query")
		case kind == "FuncCall":
			firstErr = checkFunction(node)
		case kind == "RangeVar":
			firstErr = checkTable(node, allowed, ctes)
		}
	})
	if firstErr != nil {
		return "", firstErr
	}

	out, err := pg_query.Deparse(tree)
	if err != nil {
		return "", reject("SQL could not be regenerated: %v", err)
	}
	return out, nil
}

func checkRoot(root map[string]any) error {
	if inner, ok := root["SelectStmt"]; ok && inner != nil {
		return nil
	}
	if ex, ok := root["ExplainStmt"].(map[string]any); ok {
		if opts, _ := ex["options"].([]any); len(opts) > 0 {
			return reject("EXPLAIN options and ANALYZE are not allowed")
		}
		if q, _ := ex["query"].(map[string]any); q != nil {
			if _, isSelect := q["SelectStmt"]; isSelect {
				return nil
			}
		}
	}
	return reject("Only SELECT statements are allowed")
}

func checkFunction(node map[string]any) error {
	names, _ := node["funcname"].([]any)
	if len(names) == 0 {
		return nil
	}
	last, _ := names[len(names)-1].(map[string]any)
	str, _ := last["String"].(map[string]any)
	name, _ := str["sval"].(string)
	name = strings.ToLower(name)
	if deniedFunctions[name] {
		return reject("Function %s is not allowed", name)
	}
	for _, p := range deniedFunctionPrefixes {
		if strings.HasPrefix(name, p) {
			return reject("Function %s is not allowed", name)
		}
	}
	return nil
}

func checkTable(node map[string]any, allowed, ctes map[string]bool) error {
	schema, _ := node["schemaname"].(string)
	schema = strings.ToLower(schema)
	name, _ := node["relname"].(string)
	name = strings.ToLower(name)
	if schema != "" {
		if deniedSchemas[schema] || !allowed[schema] {
			return reject("Schema %s is not allowed", schema)
		}
		return nil
	}
	if !ctes[name] && (strings.HasPrefix(name, "pg_") || deniedSchemas[name]) {
		return reject("Table %s is not allowed", name)
	}
	return nil
}

// walk visits every JSON node. A parse-tree node is an object with a single
// key naming its kind, for example {"RangeVar": {...}}.
func walk(v any, fn func(kind string, node map[string]any)) {
	switch x := v.(type) {
	case map[string]any:
		for key, val := range x {
			if child, ok := val.(map[string]any); ok {
				fn(key, child)
			}
			walk(val, fn)
		}
	case []any:
		for _, item := range x {
			walk(item, fn)
		}
	}
}

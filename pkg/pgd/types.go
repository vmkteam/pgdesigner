package pgd

import "strings"

// NormalizeType converts common PostgreSQL type aliases to canonical form.
func NormalizeType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "int2":
		return "smallint"
	case "int4":
		return "integer"
	case "int8":
		return "bigint"
	case "float4":
		return "real"
	case "float8":
		return "double precision"
	case "bool":
		return "boolean"
	case "character varying":
		return "varchar"
	case "character":
		return "char"
	case "bit varying":
		return "varbit"
	case "timestamp with time zone":
		return "timestamptz"
	case "timestamp without time zone":
		return "timestamp"
	case "time with time zone":
		return "timetz"
	case "time without time zone":
		return "time"
	case "int4[]":
		return "integer[]"
	case "int2[]":
		return "smallint[]"
	case "int8[]":
		return "bigint[]"
	case "bool[]":
		return "boolean[]"
	default:
		return t
	}
}

// ColRefsFromNames converts a string slice to a ColRef slice.
func ColRefsFromNames(names []string) []ColRef {
	refs := make([]ColRef, len(names))
	for i, n := range names {
		refs[i] = ColRef{Name: n}
	}
	return refs
}

// IsExpression reports whether s looks like an SQL expression rather than a plain column name.
func IsExpression(s string) bool {
	c := strings.TrimSpace(s)
	lower := strings.ToLower(c)
	if lower == "true" || lower == "false" || lower == "null" {
		return true
	}
	return strings.ContainsAny(c, "()+->:")
}

// FKActionFromPGCode converts a single-letter PostgreSQL FK action code to pgd internal form.
func FKActionFromPGCode(code string) string {
	switch code {
	case "c":
		return "cascade"
	case "n":
		return "set-null"
	case "d":
		return "set-default"
	case "r":
		return "restrict"
	case "a":
		return "no action"
	default:
		return "no action"
	}
}

// NeedsLength reports whether the type accepts a length parameter.
func NeedsLength(t string) bool {
	switch strings.ToLower(t) {
	case "varchar", "character varying", "char", "character", "bit", "varbit", "bit varying":
		return true
	}
	return false
}

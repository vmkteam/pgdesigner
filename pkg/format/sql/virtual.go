package sql

import (
	"regexp"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// virtualMarker wraps the expression of a PG18 virtual generated column, see markVirtualGenerated.
const virtualMarker = "__pgdesigner_virtual__"

var generatedAsRe = regexp.MustCompile(`(?i)^GENERATED\s+ALWAYS\s+AS\s*\(`)

// markVirtualGenerated rewrites PG18 virtual generated columns, which the PG17 parser
// rejects, into the STORED form with the expression wrapped in a marker call:
//
//	GENERATED ALWAYS AS (expr) [VIRTUAL] → GENERATED ALWAYS AS (__pgdesigner_virtual__(expr)) STORED
//
// pg_dump 18 writes virtual columns without the VIRTUAL keyword. String literals,
// quoted identifiers, comments and dollar-quoted bodies are left untouched.
func markVirtualGenerated(sql string) string {
	var b strings.Builder
	last := 0 // sql[:last] is already written to b
	for i := 0; i < len(sql); {
		if end, ok := skipQuoted(sql, i); ok {
			i = end
			continue
		}
		if (sql[i] != 'G' && sql[i] != 'g') || (i > 0 && isIdentChar(sql[i-1])) {
			i++
			continue
		}
		m := generatedAsRe.FindStringIndex(sql[i:])
		if m == nil {
			i++
			continue
		}
		open := i + m[1] - 1
		closing := matchParen(sql, open)
		if closing < 0 {
			break
		}
		kindStart := closing + 1
		for kindStart < len(sql) && isSpace(sql[kindStart]) {
			kindStart++
		}
		kind := strings.ToUpper(identAt(sql, kindStart))
		if kind == "STORED" {
			i = closing + 1
			continue
		}

		b.WriteString(sql[last : open+1])
		b.WriteString(virtualMarker + "(")
		b.WriteString(sql[open+1 : closing])
		b.WriteString(")) STORED")
		last = closing + 1
		if kind == "VIRTUAL" {
			last = kindStart + len(kind)
		}
		i = last
	}
	if last == 0 {
		return sql
	}
	b.WriteString(sql[last:])
	return b.String()
}

// convertGenerated converts a generated column expression, unwrapping the virtual marker.
func convertGenerated(expr *pg.Node) *pgd.Generated {
	if expr == nil {
		return nil
	}
	if fc := expr.GetFuncCall(); fc != nil && funcName(fc.Funcname) == virtualMarker && len(fc.Args) == 1 {
		return &pgd.Generated{Expression: nodeToSQL(fc.Args[0]), Stored: "false"}
	}
	return &pgd.Generated{Expression: nodeToSQL(expr), Stored: "true"}
}

// matchParen returns the index of the parenthesis closing the one at open, or -1.
func matchParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); {
		if end, ok := skipQuoted(s, i); ok {
			i = end
			continue
		}
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// skipQuoted reports whether a string literal, quoted identifier, comment or
// dollar-quoted string starts at s[i], and returns the index just after it.
func skipQuoted(s string, i int) (int, bool) {
	rest := s[i:]
	switch {
	case rest[0] == '\'' || rest[0] == '"':
		q := rest[0]
		for j := 1; j < len(rest); j++ {
			if rest[j] != q {
				continue
			}
			if j+1 < len(rest) && rest[j+1] == q { // doubled quote
				j++
				continue
			}
			return i + j + 1, true
		}
		return len(s), true
	case strings.HasPrefix(rest, "--"):
		if j := strings.IndexByte(rest, '\n'); j >= 0 {
			return i + j + 1, true
		}
		return len(s), true
	case strings.HasPrefix(rest, "/*"):
		if j := strings.Index(rest[2:], "*/"); j >= 0 {
			return i + 2 + j + 2, true
		}
		return len(s), true
	case rest[0] == '$' && (i == 0 || !isIdentChar(s[i-1])):
		tag := dollarTag(rest)
		if tag == "" {
			return 0, false
		}
		if j := strings.Index(rest[len(tag):], tag); j >= 0 {
			return i + len(tag) + j + len(tag), true
		}
		return len(s), true
	}
	return 0, false
}

// dollarTag returns the opening dollar-quote tag ($$ or $name$) at the start of s, or "".
func dollarTag(s string) string {
	for j := 1; j < len(s); j++ {
		switch c := s[j]; {
		case c == '$':
			return s[:j+1]
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80:
		case c >= '0' && c <= '9' && j > 1:
		default:
			return ""
		}
	}
	return ""
}

// identAt returns the unquoted identifier or keyword starting at s[i].
func identAt(s string, i int) string {
	j := i
	for j < len(s) && isIdentChar(s[j]) {
		j++
	}
	return s[i:j]
}

func isIdentChar(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c >= 0x80
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

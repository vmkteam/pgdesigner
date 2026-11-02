package sql

import (
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	pgquery "github.com/wasilibs/go-pgquery"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// virtualMarker wraps the expression of a PG18 virtual generated column, see markVirtualGenerated.
const virtualMarker = "__pgdesigner_virtual__"

// markVirtualGenerated rewrites PG18 virtual generated columns, which the PG17 parser
// rejects, into the STORED form with the expression wrapped in a marker call:
//
//	GENERATED ALWAYS AS (expr) [VIRTUAL] → GENERATED ALWAYS AS (__pgdesigner_virtual__(expr)) STORED
//
// pg_dump 18 writes virtual columns without the VIRTUAL keyword. The SQL is tokenized by
// the PostgreSQL lexer, so string literals, comments and function bodies are left untouched.
func markVirtualGenerated(sql string) string {
	if !strings.Contains(strings.ToUpper(sql), "GENERATED") {
		return sql // lexing a large dump is not free
	}
	scan, err := pgquery.Scan(sql)
	if err != nil {
		return sql // the parser reports the error
	}
	toks := scan.Tokens

	var b strings.Builder
	last := 0 // sql[:last] is already written to b
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Token != pg.Token_GENERATED || toks[i+1].Token != pg.Token_ALWAYS ||
			toks[i+2].Token != pg.Token_AS || toks[i+3].Token != pg.Token_ASCII_40 {
			continue
		}
		open := toks[i+3]
		closing := matchParen(toks, i+3)
		if closing < 0 {
			break
		}
		next := nextToken(toks, closing)
		if next != nil && next.Token == pg.Token_STORED {
			i = closing
			continue
		}

		b.WriteString(sql[last:open.End])
		b.WriteString(virtualMarker + "(")
		b.WriteString(sql[open.End:toks[closing].Start])
		b.WriteString(")) STORED")
		last = int(toks[closing].End)
		if next != nil && next.Token == pg.Token_IDENT && strings.EqualFold(sql[next.Start:next.End], "virtual") {
			last = int(next.End)
		}
		i = closing
	}
	if last == 0 {
		return sql
	}
	b.WriteString(sql[last:])
	return b.String()
}

// matchParen returns the index of the token closing the parenthesis at toks[open], or -1.
func matchParen(toks []*pg.ScanToken, open int) int {
	depth := 0
	for i := open; i < len(toks); i++ {
		switch toks[i].Token {
		case pg.Token_ASCII_40:
			depth++
		case pg.Token_ASCII_41:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// nextToken returns the first token after toks[i] that is not a comment, or nil.
func nextToken(toks []*pg.ScanToken, i int) *pg.ScanToken {
	for _, t := range toks[i+1:] {
		if t.Token != pg.Token_SQL_COMMENT && t.Token != pg.Token_C_COMMENT {
			return t
		}
	}
	return nil
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

package pgre

import (
	"fmt"
	"regexp"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	pgquery "github.com/wasilibs/go-pgquery"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

var typeModsRe = regexp.MustCompile(`\((\d+)(?:,(\d+))?\)`)

// convertTable converts a pg_catalog table row + its columns/constraints/indexes to pgd types.
func (i *introspector) convertTable(pt pgTable) (*pgd.Table, []pgd.Index, error) {
	t := &pgd.Table{
		Name:    pt.TableName,
		Comment: pt.Comment,
	}
	if pt.RelPersistence == "u" {
		t.Unlogged = "true"
	}
	if pt.Tablespace != "" {
		t.Tablespace = pt.Tablespace
	}

	// Columns
	cols, err := i.queryColumns(pt.OID)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range cols {
		t.Columns = append(t.Columns, convertColumn(c))
	}

	// Constraints
	cons, err := i.queryConstraints(pt.OID)
	if err != nil {
		return nil, nil, err
	}
	for _, c := range cons {
		switch c.ConType {
		case "p": // PK
			t.PK = &pgd.PrimaryKey{
				Name:    c.Name,
				Columns: pgd.ColRefsFromNames(i.queryColumnNames(pt.OID, c.ConKey)),
			}
		case "u": // UNIQUE
			u := pgd.Unique{
				Name:    c.Name,
				Columns: pgd.ColRefsFromNames(i.queryColumnNames(pt.OID, c.ConKey)),
			}
			if strings.Contains(c.Def, "NULLS NOT DISTINCT") {
				u.NullsDistinct = "false"
			}
			t.Uniques = append(t.Uniques, u)
		case "c": // CHECK
			t.Checks = append(t.Checks, pgd.Check{
				Name:       c.Name,
				Expression: c.CheckExpr,
			})
		case "x": // EXCLUDE
			if ex := parseExcludeConstraintDef(c.Name, c.Def); ex != nil {
				t.Excludes = append(t.Excludes, *ex)
			}
		case "f": // FK
			refSchema, refTable := i.queryRefTableName(c.ConfRelID)
			toTable := refTable
			if refSchema != "" && refSchema != "public" {
				toTable = refSchema + "." + refTable
			}
			fk := pgd.ForeignKey{
				Name:     c.Name,
				ToTable:  toTable,
				OnDelete: pgd.FKActionFromPGCode(c.ConfDelType),
				OnUpdate: pgd.FKActionFromPGCode(c.ConfUpdType),
			}
			if c.Deferrable {
				fk.Deferrable = "true"
			}
			srcCols := i.queryColumnNames(pt.OID, c.ConKey)
			refCols := i.queryColumnNames(c.ConfRelID, c.ConfKey)
			for j := range srcCols {
				ref := ""
				if j < len(refCols) {
					ref = refCols[j]
				}
				fk.Columns = append(fk.Columns, pgd.FKCol{Name: srcCols[j], References: ref})
			}
			t.FKs = append(t.FKs, fk)
		}
	}

	// Indexes
	idxRows, err := i.queryIndexes(pt.OID)
	if err != nil {
		return nil, nil, err
	}
	var indexes []pgd.Index
	for _, idx := range idxRows {
		if pgdIdx := convertIndex(pt.TableName, idx); pgdIdx != nil {
			indexes = append(indexes, *pgdIdx)
		}
	}

	return t, indexes, nil
}

func convertColumn(c pgColumn) pgd.Column {
	col := pgd.Column{
		Name: c.Name,
		Type: pgd.NormalizeType(parseBaseType(c.Type)),
	}

	// Parse length/precision from format_type output
	if l, p, s := parseTypeMods(c.Type); l > 0 || p > 0 {
		if pgd.NeedsLength(strings.TrimSuffix(col.Type, "[]")) {
			col.Length = l
		} else {
			col.Precision = p
			col.Scale = s
		}
	}

	if c.NotNull {
		col.Nullable = "false"
	}

	// Default — skip identity-generated defaults; keep nextval for sequence-owned columns
	if c.DefaultValue != "" && c.AttIdentity == "" {
		col.Default = normalizeDefault(c.DefaultValue)
	}

	if c.AttIdentity != "" {
		gen := "by-default"
		if c.AttIdentity == "a" {
			gen = "always"
		}
		col.Identity = &pgd.Identity{Generated: gen}
	}

	if c.AttGenerated == "s" {
		col.Generated = &pgd.Generated{
			Expression: c.DefaultValue,
			Stored:     "true",
		}
		col.Default = "" // generated columns don't have separate default
	}

	if c.AttCompression != "" {
		switch c.AttCompression {
		case "l":
			col.Compression = "lz4"
		case "p":
			col.Compression = "pglz"
		}
	}

	if c.Collation != "" && c.Collation != "default" {
		col.Collation = c.Collation
	}

	if c.Comment != "" {
		col.Comment = c.Comment
	}

	if c.AttStorage != "" && c.AttStorage != c.TypeStorage {
		switch c.AttStorage {
		case "p":
			col.Storage = "plain"
		case "e":
			col.Storage = "external"
		case "m":
			col.Storage = "main"
		case "x":
			col.Storage = "extended"
		}
	}

	return col
}

// normalizeDefault cleans PG default expressions for pgd model.
// "'G'::mpaa_rating" → "'G'", "('now'::text)::date" → "CURRENT_DATE".
func normalizeDefault(s string) string {
	// Normalize now() variants
	switch s {
	case "('now'::text)::date", "CURRENT_DATE":
		return "CURRENT_DATE"
	case "now()", "CURRENT_TIMESTAMP":
		return "now()"
	}
	// Strip public. schema prefix from function calls
	s = strings.ReplaceAll(s, "public.", "")
	// Strip type cast: 'value'::type → 'value'
	if idx := strings.Index(s, "::"); idx > 0 && strings.HasPrefix(s, "'") {
		return s[:idx]
	}
	return s
}

// parseBaseType extracts base type from format_type output.
// "character varying(255)" → "varchar"
// "timestamp with time zone" → "timestamptz"
// "integer[]" → "integer[]"
// "\"Flag\"" → "Flag" (domain types are double-quoted by format_type)
func parseBaseType(s string) string {
	// Strip double quotes (domain/custom types from format_type)
	s = strings.ReplaceAll(s, "\"", "")

	// Remove array suffix temporarily
	arraySuffix := ""
	if strings.HasSuffix(s, "[]") {
		arraySuffix = "[]"
		s = s[:len(s)-2]
	}

	// Remove parenthesized modifiers
	if idx := strings.IndexByte(s, '('); idx > 0 {
		s = s[:idx]
	}

	return strings.TrimSpace(s) + arraySuffix
}

// parseTypeMods extracts length/precision/scale from format_type output.
// "character varying(255)" → 255, 0, 0
// "numeric(10,2)" → 0, 10, 2
func parseTypeMods(s string) (length, precision, scale int) {
	m := typeModsRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0, 0, 0
	}

	n := atoi(m[1])
	if len(m) >= 3 && m[2] != "" {
		return 0, n, atoi(m[2])
	}
	return n, 0, 0
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// convertIndex parses pg_get_indexdef output into pgd.Index.
func convertIndex(tableName string, idx pgIndex) *pgd.Index {
	out := &pgd.Index{
		Name:  idx.IndexName,
		Table: tableName,
	}
	if idx.IsUnique {
		out.Unique = "true"
	}
	if idx.Method != "btree" {
		out.Using = idx.Method
	}
	if idx.Predicate != "" {
		out.Where = &pgd.WhereClause{Value: idx.Predicate}
	}

	// Parse columns and WITH params from index definition
	def, withParams := splitIndexWith(idx.Def)
	if len(withParams) > 0 {
		out.With = &pgd.With{Params: withParams}
	}
	for _, elem := range parseIndexColumns(def) {
		if pgd.IsExpression(elem.Name) {
			out.Expressions = append(out.Expressions, pgd.Expression{Value: elem.Name})
		} else {
			elem.Name = strings.Trim(elem.Name, `"`)
			out.Columns = append(out.Columns, elem)
		}
	}

	return out
}

// parseExcludeConstraintDef parses pg_get_constraintdef output for EXCLUDE constraints
// using pg_query to get correct AST parsing of expressions and WHERE clauses.
func parseExcludeConstraintDef(name, def string) *pgd.Exclude {
	// Wrap in ALTER TABLE to make it parseable
	sql := fmt.Sprintf(`ALTER TABLE "t" ADD CONSTRAINT %q %s;`, name, def)
	tree, err := pgquery.Parse(sql)
	if err != nil || len(tree.Stmts) == 0 {
		return &pgd.Exclude{Name: name}
	}
	alter := tree.Stmts[0].Stmt.GetAlterTableStmt()
	if alter == nil || len(alter.Cmds) == 0 {
		return &pgd.Exclude{Name: name}
	}
	cmd := alter.Cmds[0].GetAlterTableCmd()
	if cmd == nil || cmd.Def == nil {
		return &pgd.Exclude{Name: name}
	}
	con := cmd.Def.GetConstraint()
	if con == nil {
		return &pgd.Exclude{Name: name}
	}

	ex := pgd.Exclude{
		Name:  name,
		Using: con.AccessMethod,
	}
	for _, node := range con.Exclusions {
		list := node.GetList()
		if list == nil || len(list.Items) < 2 {
			continue
		}
		var elem pgd.ExcludeElement
		if ie := list.Items[0].GetIndexElem(); ie != nil {
			if ie.Name != "" {
				elem.Column = ie.Name
			} else if ie.Expr != nil {
				elem.Expression = nodeToSQL(ie.Expr)
			}
		}
		if opList := list.Items[1].GetList(); opList != nil {
			for _, op := range opList.Items {
				if s := op.GetString_(); s != nil {
					elem.With = s.Sval
				}
			}
		}
		ex.Elements = append(ex.Elements, elem)
	}
	if con.WhereClause != nil {
		ex.Where = &pgd.WhereClause{Value: nodeToSQL(con.WhereClause)}
	}
	return &ex
}

// nodeToSQL deparses a pg_query node back to SQL text.
func nodeToSQL(n *pg.Node) string {
	tree := &pg.ParseResult{
		Stmts: []*pg.RawStmt{{
			Stmt: &pg.Node{Node: &pg.Node_SelectStmt{
				SelectStmt: &pg.SelectStmt{
					TargetList: []*pg.Node{{
						Node: &pg.Node_ResTarget{
							ResTarget: &pg.ResTarget{Val: n},
						},
					}},
				},
			}},
		}},
	}
	result, err := pgquery.Deparse(tree)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(result, "SELECT ")
}

// splitIndexWith splits pg_get_indexdef output into the index definition (without WITH)
// and extracted storage parameters. e.g. "CREATE INDEX ... (...) WITH (fastupdate='true')"
// → "CREATE INDEX ... (...)", [{fastupdate, true}]
func splitIndexWith(def string) (string, []pgd.WithParam) {
	idx := strings.Index(strings.ToUpper(def), ") WITH (")
	if idx < 0 {
		return def, nil
	}
	withStart := idx + len(") WITH (")
	withEnd := strings.Index(def[withStart:], ")")
	if withEnd < 0 {
		return def, nil
	}
	withStr := def[withStart : withStart+withEnd]
	rest := def[withStart+withEnd+1:]

	var params []pgd.WithParam
	for _, p := range strings.Split(withStr, ",") {
		p = strings.TrimSpace(p)
		if eq := strings.IndexByte(p, '='); eq > 0 {
			params = append(params, pgd.WithParam{
				Name:  strings.TrimSpace(p[:eq]),
				Value: strings.Trim(strings.TrimSpace(p[eq+1:]), "'"),
			})
		}
	}

	return def[:idx+1] + rest, params
}

// parseIndexColumns extracts columns with sort direction from pg_get_indexdef output.
func parseIndexColumns(def string) []pgd.ColRef {
	onIdx := strings.Index(strings.ToUpper(def), " ON ")
	if onIdx < 0 {
		return nil
	}
	start := strings.IndexByte(def[onIdx:], '(')
	if start < 0 {
		return nil
	}
	start += onIdx
	depth := 1
	end := start + 1
	for end < len(def) && depth > 0 {
		switch def[end] {
		case '(':
			depth++
		case ')':
			depth--
		}
		end++
	}
	if depth != 0 {
		return nil
	}
	inner := def[start+1 : end-1]

	parts := splitTopLevel(inner, ',')
	var cols []pgd.ColRef
	for _, p := range parts {
		col := strings.TrimSpace(p)
		var ref pgd.ColRef
		if strings.HasSuffix(col, " NULLS FIRST") {
			ref.Nulls = "first"
			col = strings.TrimSuffix(col, " NULLS FIRST")
		} else if strings.HasSuffix(col, " NULLS LAST") {
			ref.Nulls = "last"
			col = strings.TrimSuffix(col, " NULLS LAST")
		}
		if strings.HasSuffix(col, " DESC") {
			ref.Order = "desc"
			col = strings.TrimSuffix(col, " DESC")
		} else {
			col = strings.TrimSuffix(col, " ASC")
		}
		col = strings.TrimSpace(col)
		if col != "" {
			ref.Name = col
			cols = append(cols, ref)
		}
	}
	return cols
}

// splitTopLevel splits a string by sep, but only at the top level (not inside parentheses).
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	depth := 0
	start := 0
	for i := range len(s) {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func convertSequence(s pgSequence) pgd.Sequence {
	seq := pgd.Sequence{
		Name:   s.Name,
		Schema: s.Schema,
	}
	if s.Start != 1 {
		seq.Start = s.Start
	}
	if s.Increment != 1 {
		seq.Increment = s.Increment
	}
	if s.Cache != 1 {
		seq.Cache = s.Cache
	}
	if s.Cycle {
		seq.Cycle = "true"
	}
	return seq
}

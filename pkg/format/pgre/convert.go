package pgre

import (
	"regexp"
	"strings"

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
			t.Uniques = append(t.Uniques, pgd.Unique{
				Name:    c.Name,
				Columns: pgd.ColRefsFromNames(i.queryColumnNames(pt.OID, c.ConKey)),
			})
		case "c": // CHECK
			t.Checks = append(t.Checks, pgd.Check{
				Name:       c.Name,
				Expression: c.CheckExpr,
			})
		case "x": // EXCLUDE
			t.Excludes = append(t.Excludes, pgd.Exclude{
				Name: c.Name,
			})
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
		if pgd.NeedsLength(col.Type) {
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

	if c.AttStorage != "" {
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

	// Parse columns from index definition
	cols := parseIndexColumns(idx.Def)
	for _, c := range cols {
		if pgd.IsExpression(c) {
			out.Expressions = append(out.Expressions, pgd.Expression{Value: c})
		} else {
			out.Columns = append(out.Columns, pgd.ColRef{Name: strings.Trim(c, `"`)})
		}
	}

	return out
}

// parseIndexColumns extracts column names from pg_get_indexdef output.
// "CREATE INDEX idx ON public.table USING btree (col1, col2)" → ["col1", "col2"]
func parseIndexColumns(def string) []string {
	// Find content between last ( and last )
	start := strings.LastIndex(def, "(")
	end := strings.LastIndex(def, ")")
	if start < 0 || end < 0 || end <= start {
		return nil
	}

	inner := def[start+1 : end]
	// Remove WHERE clause if present
	if idx := strings.Index(strings.ToUpper(inner), " WHERE "); idx > 0 {
		inner = inner[:idx]
	}

	parts := strings.Split(inner, ",")
	var cols []string
	for _, p := range parts {
		col := strings.TrimSpace(p)
		// Remove sort direction
		col = strings.TrimSuffix(col, " ASC")
		col = strings.TrimSuffix(col, " DESC")
		col = strings.TrimSuffix(col, " NULLS FIRST")
		col = strings.TrimSuffix(col, " NULLS LAST")
		col = strings.TrimSpace(col)
		if col != "" {
			cols = append(cols, col)
		}
	}
	return cols
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

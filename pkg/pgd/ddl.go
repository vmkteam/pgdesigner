package pgd

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// ddlWriter wraps io.Writer with sticky error semantics.
// After the first write error, all subsequent writes are no-ops.
type ddlWriter struct {
	w   io.Writer
	err error
}

// P writes formatted output (like fmt.Fprintf). No-op after first error.
func (d *ddlWriter) P(format string, args ...any) {
	if d.err != nil {
		return
	}
	_, d.err = fmt.Fprintf(d.w, format, args...)
}

// S writes a raw string. No-op after first error.
func (d *ddlWriter) S(s string) {
	if d.err != nil {
		return
	}
	_, d.err = io.WriteString(d.w, s)
}

// Nl writes a blank line (block separator).
func (d *ddlWriter) Nl() { d.S("\n\n") }

// If writes formatted output only when cond is true.
func (d *ddlWriter) If(cond bool, format string, args ...any) {
	if cond {
		d.P(format, args...)
	}
}

// Err returns the first error encountered.
func (d *ddlWriter) Err() error { return d.err }

// WriteDDL writes a full DDL script for the project to w.
// Output order follows dependency topology:
//
//  1. CREATE ROLE
//  2. CREATE EXTENSION
//  3. CREATE TYPE (enum, composite, range)
//  4. CREATE DOMAIN
//  5. CREATE SCHEMA (non-public)
//  6. CREATE SEQUENCE
//  7. CREATE TABLE (columns, PK, UNIQUE, CHECK — no FK)
//  8. ALTER TABLE ENABLE ROW LEVEL SECURITY
//  9. CREATE INDEX
//  10. ALTER TABLE ADD FOREIGN KEY
//  11. COMMENT ON (table + column)
//  12. ALTER COLUMN SET STORAGE
//  13. CREATE FUNCTION / PROCEDURE
//  14. CREATE MATERIALIZED VIEW / VIEW
//  15. CREATE INDEX (on mat views)
//  16. CREATE TRIGGER
//  17. CREATE POLICY
//  18. COMMENT ON (non-table)
//  19. GRANT
func WriteDDL(w io.Writer, p *Project) error {
	d := &ddlWriter{w: w}

	// Phase 0: roles
	for i := range p.Roles {
		d.writeRole(&p.Roles[i])
	}

	// Phase 1: extensions
	for i := range p.Extensions {
		d.writeExtension(&p.Extensions[i])
	}

	// Phase 2: types
	if p.Types != nil {
		for i := range p.Types.Enums {
			d.writeEnum(&p.Types.Enums[i])
		}
		for i := range p.Types.Composites {
			d.writeComposite(&p.Types.Composites[i])
		}
		for i := range p.Types.Ranges {
			d.writeRange(&p.Types.Ranges[i])
		}
		for i := range p.Types.Domains {
			d.writeDomain(&p.Types.Domains[i])
		}
	}

	// Phase 3: CREATE SCHEMA for all non-public schemas
	for _, name := range collectSchemas(p) {
		d.P("CREATE SCHEMA IF NOT EXISTS %s;\n\n", QuoteIdent(name))
	}

	// Phase 4: sequences (skip identity-owned)
	identitySeqs := identitySeqNames(p)
	for i := range p.Sequences {
		if identitySeqs[p.Sequences[i].Name] {
			continue
		}
		d.writeSequence(&p.Sequences[i])
	}

	// collect mat view names for deferred index creation
	mvNames := matViewNames(p)

	// Phases 5–7.6: tables, RLS, indexes, FKs, comments, storage
	d.writeSchemaObjects(p, mvNames)

	// Phase 8: functions (before views — views may call functions/aggregates)
	for _, f := range sortFunctionsByDeps(p.Functions) {
		d.writeFunction(f)
	}

	// Phase 9: mat views first (views may depend on them), then views
	if p.Views != nil {
		d.writeMatViewsSorted(p.Views.MatViews)
		for i := range p.Views.Views {
			d.writeView(&p.Views.Views[i])
		}
	}

	// Phase 9.5: CREATE INDEX on mat views (deferred from Phase 6)
	d.writeMatViewIndexes(p, mvNames)

	// Phase 10: triggers
	for i := range p.Triggers {
		d.writeTrigger(&p.Triggers[i])
	}

	// Phase 11: RLS policies
	for i := range p.Policies {
		d.writePolicy(&p.Policies[i])
	}

	// Phase 12: COMMENT ON (non-table/column)
	for i := range p.Comments {
		d.writeComment(&p.Comments[i])
	}

	// Phase 13: GRANT privileges
	if p.Grants != nil {
		for i := range p.Grants.Grants {
			d.writeGrant(&p.Grants.Grants[i])
		}
		for i := range p.Grants.GrantRoles {
			d.writeGrantRole(&p.Grants.GrantRoles[i])
		}
	}

	return d.Err()
}

// writeSchemaObjects writes tables, RLS, indexes, FKs, comments, and column storage for all schemas.
func (d *ddlWriter) writeSchemaObjects(p *Project, mvNames map[string]bool) {
	partChildren := CollectPartitionChildren(p)

	skipSets := make([]map[string]bool, len(p.Schemas))
	for i := range p.Schemas {
		skipSets[i] = skipSet(&p.Schemas[i])
	}

	// CREATE TABLE
	for i := range p.Schemas {
		s := &p.Schemas[i]
		skip := skipSets[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			t := &s.Tables[j]
			if skip[t.Name] || t.PartitionOf != "" {
				continue
			}
			d.writeTable(q, t)
		}
	}

	// ALTER TABLE ENABLE ROW LEVEL SECURITY
	for i := range p.Schemas {
		s := &p.Schemas[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			d.writeRLS(q, &s.Tables[j])
		}
	}

	// CREATE INDEX (skip indexes on mat views and partition children)
	for i := range p.Schemas {
		s := &p.Schemas[i]
		skip := skipSets[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Indexes {
			tbl := s.Indexes[j].Table
			if skip[tbl] || mvNames[tbl] || partChildren[tbl] {
				continue
			}
			d.writeIndex(q, &s.Indexes[j])
		}
	}

	// ALTER TABLE ADD FK (skip partition children)
	for i := range p.Schemas {
		s := &p.Schemas[i]
		skip := skipSets[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			t := &s.Tables[j]
			if skip[t.Name] || partChildren[t.Name] || t.PartitionOf != "" {
				continue
			}
			for k := range t.FKs {
				d.writeFK(q, t.Name, &t.FKs[k])
			}
		}
	}

	// COMMENT ON (table + column)
	for i := range p.Schemas {
		s := &p.Schemas[i]
		skip := skipSets[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			if skip[s.Tables[j].Name] {
				continue
			}
			d.writeTableComments(q, &s.Tables[j])
		}
	}

	// ALTER COLUMN SET STORAGE
	for i := range p.Schemas {
		s := &p.Schemas[i]
		skip := skipSets[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			if skip[s.Tables[j].Name] {
				continue
			}
			d.writeColumnStorage(q, &s.Tables[j])
		}
	}
}

// writeMatViewIndexes writes CREATE INDEX statements for materialized views.
func (d *ddlWriter) writeMatViewIndexes(p *Project, mvNames map[string]bool) {
	for i := range p.Schemas {
		s := &p.Schemas[i]
		q := SchemaQualifier(s.Name)
		for j := range s.Indexes {
			if !mvNames[s.Indexes[j].Table] {
				continue
			}
			d.writeIndex(q, &s.Indexes[j])
		}
	}
}

// collectSchemas returns all non-public schema names from schemas, sequences, views, mat views.
func collectSchemas(p *Project) []string {
	seen := make(map[string]bool)
	for _, s := range p.Schemas {
		if s.Name != "" && s.Name != "public" {
			seen[s.Name] = true
		}
	}
	for _, seq := range p.Sequences {
		if seq.Schema != "" && seq.Schema != "public" {
			seen[seq.Schema] = true
		}
	}
	if p.Views != nil {
		for _, v := range p.Views.Views {
			if v.Schema != "" && v.Schema != "public" {
				seen[v.Schema] = true
			}
		}
		for _, mv := range p.Views.MatViews {
			if mv.Schema != "" && mv.Schema != "public" {
				seen[mv.Schema] = true
			}
		}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// matViewNames returns set of materialized view names for deferred index creation.
func matViewNames(p *Project) map[string]bool {
	names := make(map[string]bool)
	if p.Views == nil {
		return names
	}
	for _, mv := range p.Views.MatViews {
		names[mv.Name] = true
	}
	return names
}

// identitySeqNames returns set of sequence names owned by identity columns.
func identitySeqNames(p *Project) map[string]bool {
	names := make(map[string]bool)
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			for _, c := range t.Columns {
				if c.Identity != nil {
					names[t.Name+"_"+c.Name+"_seq"] = true
				}
			}
		}
	}
	return names
}

// --- extensions ---

func (d *ddlWriter) writeExtension(e *Extension) {
	d.P("CREATE EXTENSION IF NOT EXISTS %s", QuoteIdent(e.Name))
	d.If(e.Schema != "", " SCHEMA %s", QuoteIdent(e.Schema))
	d.If(e.Version != "", " VERSION '%s'", e.Version)
	d.S(";\n\n")
}

// --- types ---

func (d *ddlWriter) writeEnum(e *Enum) {
	labels := make([]string, len(e.Labels))
	for i, l := range e.Labels {
		labels[i] = "'" + l + "'"
	}
	d.P("CREATE TYPE %s AS ENUM (%s);\n\n", QualifiedName(e.Schema, e.Name), strings.Join(labels, ", "))
}

func (d *ddlWriter) writeComposite(c *Composite) {
	var fields []string
	for _, f := range c.Fields {
		s := QuoteIdent(f.Name) + " " + f.Type
		if f.Length > 0 {
			s = fmt.Sprintf("%s %s(%d)", QuoteIdent(f.Name), f.Type, f.Length)
		}
		fields = append(fields, "\t"+s)
	}
	d.P("CREATE TYPE %s AS (\n%s\n);\n\n", QualifiedName(c.Schema, c.Name), strings.Join(fields, ",\n"))
}

func (d *ddlWriter) writeRange(r *Range) {
	d.P("CREATE TYPE %s AS RANGE (SUBTYPE = %s);\n\n", QualifiedName(r.Schema, r.Name), r.Subtype)
}

func (d *ddlWriter) writeDomain(dom *Domain) {
	name := QualifiedName(dom.Schema, dom.Name)
	typSpec := dom.Type
	if dom.Length > 0 {
		typSpec = fmt.Sprintf("%s(%d)", dom.Type, dom.Length)
	}
	d.P("CREATE DOMAIN %s AS %s", name, typSpec)
	d.If(dom.Collation != "", " COLLATE %s", QuoteIdent(dom.Collation))
	d.If(dom.Default != "", " DEFAULT %s", dom.Default)
	if dom.NotNull != nil {
		d.S(" NOT NULL")
	}
	for _, c := range dom.Constraints {
		if c.Name != "" {
			d.P(" CONSTRAINT %s", QuoteIdent(c.Name))
		}
		d.P(" CHECK (%s)", c.Expression)
	}
	d.S(";\n\n")
}

// --- sequences ---

func (d *ddlWriter) writeSequence(s *Sequence) {
	d.P("CREATE SEQUENCE %s", QualifiedName(s.Schema, s.Name))
	d.If(s.Type != "", " AS %s", s.Type)
	d.If(s.Increment != 0, " INCREMENT BY %d", s.Increment)
	d.If(s.Min != 0, " MINVALUE %d", s.Min)
	d.If(s.Max != 0, " MAXVALUE %d", s.Max)
	d.If(s.Start != 0, " START WITH %d", s.Start)
	d.If(s.Cache != 0, " CACHE %d", s.Cache)
	d.If(s.Cycle == "true", " CYCLE")
	d.If(s.OwnedBy != "", " OWNED BY %s", s.OwnedBy)
	d.S(";\n\n")
}

// --- views ---

func (d *ddlWriter) writeView(v *View) {
	d.S("CREATE")
	d.If(v.Recursive == "true", " RECURSIVE")
	d.P(" VIEW %s AS\n%s;\n\n", QualifiedName(v.Schema, v.Name), strings.TrimSpace(v.Query))
}

func (d *ddlWriter) writeMatView(mv *MaterializedView) {
	d.P("CREATE MATERIALIZED VIEW %s", QualifiedName(mv.Schema, mv.Name))
	d.If(mv.Tablespace != "", " TABLESPACE %s", QuoteIdent(mv.Tablespace))
	d.P(" AS\n%s", strings.TrimSpace(mv.Query))
	d.If(mv.WithData == "false", "\nWITH NO DATA")
	d.S(";\n\n")
}

func (d *ddlWriter) writeMatViewWithNoData(mv *MaterializedView) {
	d.P("CREATE MATERIALIZED VIEW %s", QualifiedName(mv.Schema, mv.Name))
	d.If(mv.Tablespace != "", " TABLESPACE %s", QuoteIdent(mv.Tablespace))
	d.P(" AS\n%s\nWITH NO DATA;\n\n", strings.TrimSpace(mv.Query))
}

// writeMatViewsSorted writes materialized views in dependency order.
func (d *ddlWriter) writeMatViewsSorted(mvs []MaterializedView) {
	if len(mvs) == 0 {
		return
	}

	written := make(map[int]bool)
	remaining := len(mvs)

	for remaining > 0 {
		progress := false
		for i := range mvs {
			if written[i] {
				continue
			}
			canWrite := true
			for j := range mvs {
				if written[j] || i == j {
					continue
				}
				q := mvs[i].Query
				referenced := strings.Contains(q, `"`+mvs[j].Name+`"`) ||
					strings.Contains(q, `.`+mvs[j].Name+` `) ||
					strings.Contains(q, `.`+mvs[j].Name+`)`)
				if referenced {
					canWrite = false
					break
				}
			}
			if canWrite {
				d.writeMatView(&mvs[i])
				written[i] = true
				remaining--
				progress = true
			}
		}
		if !progress {
			for i := range mvs {
				if !written[i] {
					d.writeMatViewWithNoData(&mvs[i])
					written[i] = true
					remaining--
				}
			}
		}
	}
}

// --- functions ---

func (d *ddlWriter) writeFunction(f *Function) {
	if f.Kind == "aggregate" {
		d.writeAggregate(f)
		return
	}

	name := QualifiedName(f.Schema, f.Name)
	keyword := "FUNCTION"
	if f.Kind == "procedure" {
		keyword = "PROCEDURE"
	}

	var args []string
	for _, a := range f.Args {
		var parts []string
		if a.Mode != "" && a.Mode != "in" {
			parts = append(parts, strings.ToUpper(a.Mode))
		}
		if a.Name != "" {
			parts = append(parts, QuoteIdent(a.Name))
		}
		parts = append(parts, a.Type)
		if a.Default != "" {
			parts = append(parts, "DEFAULT", a.Default)
		}
		args = append(args, strings.Join(parts, " "))
	}

	d.P("CREATE OR REPLACE %s %s(%s)", keyword, name, strings.Join(args, ", "))

	if f.Returns != "" {
		d.P("\nRETURNS %s", f.Returns)
	} else if f.RetTable != nil {
		var cols []string
		for _, c := range f.RetTable.Columns {
			cols = append(cols, QuoteIdent(c.Name)+" "+c.Type)
		}
		d.P("\nRETURNS TABLE(%s)", strings.Join(cols, ", "))
	}

	d.If(f.Language != "", "\nLANGUAGE %s", f.Language)
	d.If(f.Volatility != "", "\n%s", strings.ToUpper(f.Volatility))
	d.If(f.Security == "definer", "\nSECURITY DEFINER")
	d.If(f.Parallel != "" && f.Parallel != "unsafe", "\nPARALLEL %s", strings.ToUpper(f.Parallel))
	d.If(f.Strict == "true", "\nSTRICT")
	d.If(f.Leakproof == "true", "\nLEAKPROOF")
	d.If(f.Cost > 0, "\nCOST %d", f.Cost)
	d.If(f.Rows > 0, "\nROWS %d", f.Rows)
	d.If(f.Body != "", "\nAS $%s$\n%s\n$%s$", keyword, strings.TrimSpace(f.Body), keyword)
	d.S(";\n\n")
}

func (d *ddlWriter) writeAggregate(f *Function) {
	name := QualifiedName(f.Schema, f.Name)
	var args []string
	for _, a := range f.Args {
		args = append(args, a.Type)
	}
	d.P("CREATE AGGREGATE %s(%s) (\n", name, strings.Join(args, ", "))

	var opts []string
	if f.SFunc != "" {
		opts = append(opts, fmt.Sprintf("    SFUNC = %s", f.SFunc))
	}
	if f.SType != "" {
		opts = append(opts, fmt.Sprintf("    STYPE = %s", f.SType))
	}
	if f.FinalFunc != "" {
		opts = append(opts, fmt.Sprintf("    FINALFUNC = %s", f.FinalFunc))
	}
	if f.CombineFunc != "" {
		opts = append(opts, fmt.Sprintf("    COMBINEFUNC = %s", f.CombineFunc))
	}
	if f.InitCond != "" {
		opts = append(opts, fmt.Sprintf("    INITCOND = '%s'", f.InitCond))
	}
	if f.SortOp != "" {
		opts = append(opts, fmt.Sprintf("    SORTOP = %s", f.SortOp))
	}
	d.S(strings.Join(opts, ",\n"))
	d.S("\n);\n\n")
}

// --- trigger ---

func (d *ddlWriter) writeTrigger(t *Trigger) {
	if t.Constraint == "true" {
		d.S("CREATE CONSTRAINT TRIGGER ")
	} else {
		d.S("CREATE TRIGGER ")
	}
	d.P("%s\n", QuoteIdent(t.Name))

	timing := strings.ToUpper(strings.ReplaceAll(t.Timing, "-", " "))
	events := strings.ToUpper(strings.ReplaceAll(t.Events, ",", " OR "))
	d.P("\t%s %s", timing, events)
	d.If(t.UpdateOf != "", " OF %s", t.UpdateOf)

	tableName := QuoteIdent(t.Table)
	if t.Schema != "" && t.Schema != "public" {
		tableName = QuoteIdent(t.Schema) + "." + tableName
	}
	d.P("\n\tON %s", tableName)

	forEach := "STATEMENT"
	if t.ForEach == "row" {
		forEach = "ROW"
	}
	d.P("\n\tFOR EACH %s", forEach)
	d.If(t.When != "", "\n\tWHEN (%s)", t.When)
	d.P("\n\tEXECUTE FUNCTION %s();\n\n", QuoteIdent(t.Execute.Function))
}

// --- table ---

func (d *ddlWriter) writeTable(q string, t *Table) {
	var parts []string
	for i := range t.Columns {
		parts = append(parts, "\t"+ColumnDef(&t.Columns[i]))
	}
	if t.PK != nil {
		parts = append(parts, "\t"+pkDef(t.PK))
	}
	for i := range t.Uniques {
		parts = append(parts, "\t"+uniqueDef(&t.Uniques[i]))
	}
	for i := range t.Checks {
		parts = append(parts, "\t"+checkDef(&t.Checks[i]))
	}
	for i := range t.Excludes {
		parts = append(parts, "\t"+excludeDef(&t.Excludes[i]))
	}

	suffix := ""
	if t.PartitionBy != nil {
		suffix = fmt.Sprintf("\nPARTITION BY %s (%s)",
			strings.ToUpper(t.PartitionBy.Type),
			QuotedColList(t.PartitionBy.Columns))
	}

	d.P("CREATE TABLE %s%s (\n%s\n)%s;\n\n",
		q, QuoteIdent(t.Name),
		strings.Join(parts, ",\n"),
		suffix)

	for i := range t.Partitions {
		d.writePartitionChild(q, t.Name, &t.Partitions[i])
	}
}

func (d *ddlWriter) writePartitionChild(q, parent string, p *Partition) {
	d.P("CREATE TABLE %s%s PARTITION OF %s%s\n",
		q, QuoteIdent(p.Name), q, QuoteIdent(parent))
	d.If(p.Bound != "", "    %s", p.Bound)
	if p.PartitionBy != nil {
		d.P("\n    PARTITION BY %s (%s)",
			strings.ToUpper(p.PartitionBy.Type),
			QuotedColList(p.PartitionBy.Columns))
	}
	d.S(";\n\n")

	for i := range p.Partitions {
		d.writePartitionChild(q, p.Name, &p.Partitions[i])
	}
}

// WriteTable writes CREATE TABLE DDL for a single table.
// Public API used by diff engine.
func WriteTable(w io.Writer, q string, t *Table) error {
	d := &ddlWriter{w: w}
	d.writeTable(q, t)
	return d.Err()
}

func ColumnDef(c *Column) string {
	var b strings.Builder

	b.WriteString(QuoteIdent(c.Name))
	b.WriteByte(' ')

	if isSerial(c.Type) {
		b.WriteString(strings.ToUpper(c.Type))
		b.WriteString(" NOT NULL")
		writeDefault(&b, c.Default)
		return b.String()
	}

	b.WriteString(TypeSpec(c))
	if c.Collation != "" {
		fmt.Fprintf(&b, " COLLATE %s", QuoteIdent(c.Collation))
	}
	if c.Compression != "" {
		fmt.Fprintf(&b, " COMPRESSION %s", c.Compression)
	}
	writeNotNull(&b, c.Nullable)
	if c.Generated != nil {
		writeGenerated(&b, c.Generated)
	} else {
		writeIdentity(&b, c.Identity)
		if c.Identity == nil {
			writeDefault(&b, c.Default)
		}
	}

	return b.String()
}

func TypeSpec(c *Column) string {
	raw := c.Type
	suffix := ""
	if strings.HasSuffix(raw, "[]") {
		suffix = "[]"
		raw = raw[:len(raw)-2]
	}
	t := quoteType(raw)

	base := &Column{Type: raw, Length: c.Length, Precision: c.Precision, Scale: c.Scale}
	switch {
	case isNumericWithParams(base):
		if c.Scale > 0 {
			return fmt.Sprintf("%s(%d,%d)%s", t, c.Precision, c.Scale, suffix)
		}
		return fmt.Sprintf("%s(%d)%s", t, c.Precision, suffix)
	case hasLength(base):
		return fmt.Sprintf("%s(%d)%s", t, c.Length, suffix)
	default:
		return t + suffix
	}
}

// quoteType handles schema-qualified type names: "myschema.custom_type" → "myschema"."custom_type".
func quoteType(t string) string {
	if i := strings.IndexByte(t, '.'); i >= 0 {
		return QuoteIdent(t[:i]) + "." + QuoteIdent(t[i+1:])
	}
	if needsQuoting(t) {
		return QuoteIdent(t)
	}
	return t
}

// needsQuoting returns true if a type name contains uppercase letters.
func needsQuoting(t string) bool {
	for _, r := range t {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func writeNotNull(b *strings.Builder, nullable string) {
	if nullable == "false" {
		b.WriteString(" NOT NULL")
	}
}

func writeIdentity(b *strings.Builder, id *Identity) {
	if id == nil {
		return
	}
	switch id.Generated {
	case "always":
		b.WriteString(" GENERATED ALWAYS AS IDENTITY")
	default:
		b.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
	}
	if id.Sequence != nil {
		writeIdentitySeqOpts(b, id.Sequence)
	}
}

func writeIdentitySeqOpts(b *strings.Builder, seq *IdentitySeqOpt) {
	var opts []string
	if seq.Start != 0 {
		opts = append(opts, fmt.Sprintf("START WITH %d", seq.Start))
	}
	if seq.Increment != 0 {
		opts = append(opts, fmt.Sprintf("INCREMENT BY %d", seq.Increment))
	}
	if seq.Min != 0 {
		opts = append(opts, fmt.Sprintf("MINVALUE %d", seq.Min))
	}
	if seq.Max != 0 {
		opts = append(opts, fmt.Sprintf("MAXVALUE %d", seq.Max))
	}
	if seq.Cache != 0 {
		opts = append(opts, fmt.Sprintf("CACHE %d", seq.Cache))
	}
	if seq.Cycle == "true" {
		opts = append(opts, "CYCLE")
	}
	if len(opts) > 0 {
		fmt.Fprintf(b, " (%s)", strings.Join(opts, " "))
	}
}

func writeDefault(b *strings.Builder, def string) {
	if def != "" {
		fmt.Fprintf(b, " DEFAULT %s", def)
	}
}

func (d *ddlWriter) writeTableComments(q string, t *Table) {
	if t.Comment != "" {
		d.P("COMMENT ON TABLE %s%s IS %s;\n", q, QuoteIdent(t.Name), EscapeComment(t.Comment))
	}
	for i := range t.Columns {
		c := &t.Columns[i]
		if c.Comment != "" {
			d.P("COMMENT ON COLUMN %s%s.%s IS %s;\n", q, QuoteIdent(t.Name), QuoteIdent(c.Name), EscapeComment(c.Comment))
		}
	}
	if t.Comment != "" || hasColumnComments(t) {
		d.S("\n")
	}
}

func hasColumnComments(t *Table) bool {
	for i := range t.Columns {
		if t.Columns[i].Comment != "" {
			return true
		}
	}
	return false
}

func (d *ddlWriter) writeColumnStorage(q string, t *Table) {
	for i := range t.Columns {
		c := &t.Columns[i]
		if c.Storage != "" {
			d.P("ALTER TABLE %s%s ALTER COLUMN %s SET STORAGE %s;\n",
				q, QuoteIdent(t.Name), QuoteIdent(c.Name), strings.ToUpper(c.Storage))
		}
	}
	if hasColumnStorage(t) {
		d.S("\n")
	}
}

func hasColumnStorage(t *Table) bool {
	for i := range t.Columns {
		if t.Columns[i].Storage != "" {
			return true
		}
	}
	return false
}

// EscapeComment wraps s in single quotes and escapes embedded quotes for SQL.
func EscapeComment(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func writeGenerated(b *strings.Builder, g *Generated) {
	if g == nil {
		return
	}
	fmt.Fprintf(b, " GENERATED ALWAYS AS (%s)", g.Expression)
	if g.Stored == "false" {
		b.WriteString(" VIRTUAL")
	} else {
		b.WriteString(" STORED")
	}
}

func pkDef(pk *PrimaryKey) string {
	if pk.Name != "" {
		return fmt.Sprintf("CONSTRAINT %s PRIMARY KEY(%s)", QuoteIdent(pk.Name), QuotedColList(pk.Columns))
	}
	return fmt.Sprintf("PRIMARY KEY(%s)", QuotedColList(pk.Columns))
}

func uniqueDef(u *Unique) string {
	nullsDistinct := ""
	if u.NullsDistinct == "false" {
		nullsDistinct = " NULLS NOT DISTINCT"
	}
	if u.Name != "" {
		return fmt.Sprintf("CONSTRAINT %s UNIQUE%s(%s)", QuoteIdent(u.Name), nullsDistinct, QuotedColList(u.Columns))
	}
	return fmt.Sprintf("UNIQUE%s(%s)", nullsDistinct, QuotedColList(u.Columns))
}

func checkDef(c *Check) string {
	if c.Name != "" {
		return fmt.Sprintf("CONSTRAINT %s CHECK(%s)", QuoteIdent(c.Name), c.Expression)
	}
	return fmt.Sprintf("CHECK(%s)", c.Expression)
}

func excludeDef(e *Exclude) string {
	var elems []string
	for _, el := range e.Elements {
		target := QuoteIdent(el.Column)
		if el.Expression != "" {
			target = el.Expression
		}
		elems = append(elems, fmt.Sprintf("%s WITH %s", target, el.With))
	}
	using := ""
	if e.Using != "" {
		using = fmt.Sprintf(" USING %s", e.Using)
	}
	where := ""
	if e.Where != nil && strings.TrimSpace(e.Where.Value) != "" {
		where = fmt.Sprintf(" WHERE (%s)", strings.TrimSpace(e.Where.Value))
	}
	if e.Name != "" {
		return fmt.Sprintf("CONSTRAINT %s EXCLUDE%s (%s)%s", QuoteIdent(e.Name), using, strings.Join(elems, ", "), where)
	}
	return fmt.Sprintf("EXCLUDE%s (%s)%s", using, strings.Join(elems, ", "), where)
}

// sortFunctionsByDeps orders functions so that if A's body references B's name, B comes first.
func sortFunctionsByDeps(funcs []Function) []*Function {
	if len(funcs) == 0 {
		return nil
	}

	written := make(map[int]bool)
	result := make([]*Function, 0, len(funcs))

	for range funcs {
		progress := false
		for i := range funcs {
			if written[i] {
				continue
			}
			canWrite := true
			for j := range funcs {
				if written[j] || i == j {
					continue
				}
				if strings.Contains(funcs[i].Body, funcs[j].Name) {
					canWrite = false
					break
				}
			}
			if canWrite {
				result = append(result, &funcs[i])
				written[i] = true
				progress = true
			}
		}
		if !progress {
			for i := range funcs {
				if !written[i] {
					result = append(result, &funcs[i])
				}
			}
			break
		}
		if len(result) == len(funcs) {
			break
		}
	}

	return result
}

// --- role ---

func (d *ddlWriter) writeRole(r *Role) {
	d.P("CREATE ROLE %s WITH", QuoteIdent(r.Name))
	if r.Login == "true" {
		d.S(" LOGIN")
	} else {
		d.S(" NOLOGIN")
	}
	d.If(r.Superuser == "true", " SUPERUSER")
	d.If(r.Createdb == "true", " CREATEDB")
	d.If(r.Createrole == "true", " CREATEROLE")
	d.If(r.Replication == "true", " REPLICATION")
	d.If(r.Bypassrls == "true", " BYPASSRLS")
	d.If(r.Inherit == "false", " NOINHERIT")
	d.If(r.ConnectionLimit > 0, " CONNECTION LIMIT %d", r.ConnectionLimit)
	d.If(r.ValidUntil != "", " VALID UNTIL '%s'", r.ValidUntil)
	d.S(";\n")
	for _, ir := range r.InRoles {
		d.P("GRANT %s TO %s;\n", QuoteIdent(ir.Name), QuoteIdent(r.Name))
	}
	d.S("\n")
}

// --- RLS ---

func (d *ddlWriter) writeRLS(q string, t *Table) {
	if t.RowLevelSecurity != "true" {
		return
	}
	d.P("ALTER TABLE %s%s ENABLE ROW LEVEL SECURITY;\n", q, QuoteIdent(t.Name))
	d.If(t.ForceRowLevelSecurity == "true", "ALTER TABLE %s%s FORCE ROW LEVEL SECURITY;\n", q, QuoteIdent(t.Name))
	d.S("\n")
}

// --- policy ---

func (d *ddlWriter) writePolicy(p *Policy) {
	tableName := QualifiedName(p.Schema, p.Table)
	d.P("CREATE POLICY %s ON %s", QuoteIdent(p.Name), tableName)
	d.If(p.Type != "", "\n\tAS %s", strings.ToUpper(p.Type))
	d.If(p.Command != "" && p.Command != "all", "\n\tFOR %s", strings.ToUpper(p.Command))
	if p.To != "" {
		d.S("\n\tTO ")
		for i, role := range strings.Split(p.To, ",") {
			if i > 0 {
				d.S(", ")
			}
			role = strings.TrimSpace(role)
			if role == "public" || role == "PUBLIC" {
				d.S("PUBLIC")
			} else {
				d.S(QuoteIdent(role))
			}
		}
	}
	if p.Using != nil && p.Using.Value != "" {
		d.P("\n\tUSING (%s)", p.Using.Value)
	}
	if p.WithCheck != nil && p.WithCheck.Value != "" {
		d.P("\n\tWITH CHECK (%s)", p.WithCheck.Value)
	}
	d.S(";\n\n")
}

// --- comment ---

func (d *ddlWriter) writeComment(c *Comment) {
	target := commentTarget(c)
	if target == "" {
		return
	}
	d.P("COMMENT ON %s IS %s;\n", target, EscapeComment(c.Value))
}

func commentTarget(c *Comment) string {
	switch strings.ToLower(c.On) {
	case "schema":
		return "SCHEMA " + QuoteIdent(c.Name)
	case "table":
		return "TABLE " + QualifiedName(c.Schema, c.Name)
	case "column":
		return "COLUMN " + QualifiedName(c.Schema, c.Table) + "." + QuoteIdent(c.Name)
	case "index":
		return "INDEX " + QualifiedName(c.Schema, c.Name)
	case "constraint":
		return "CONSTRAINT " + QuoteIdent(c.Name) + " ON " + QualifiedName(c.Schema, c.Table)
	case "view":
		return "VIEW " + QualifiedName(c.Schema, c.Name)
	case "materialized view":
		return "MATERIALIZED VIEW " + QualifiedName(c.Schema, c.Name)
	case "function":
		return "FUNCTION " + QualifiedName(c.Schema, c.Name)
	case "trigger":
		return "TRIGGER " + QuoteIdent(c.Name) + " ON " + QualifiedName(c.Schema, c.Table)
	case "policy":
		return "POLICY " + QuoteIdent(c.Name) + " ON " + QualifiedName(c.Schema, c.Table)
	case "sequence":
		return "SEQUENCE " + QualifiedName(c.Schema, c.Name)
	case "type":
		return "TYPE " + QualifiedName(c.Schema, c.Name)
	case "domain":
		return "DOMAIN " + QualifiedName(c.Schema, c.Name)
	case "extension":
		return "EXTENSION " + QuoteIdent(c.Name)
	case "database":
		return "DATABASE " + QuoteIdent(c.Name)
	case "role":
		return "ROLE " + QuoteIdent(c.Name)
	}
	return ""
}

// --- grant ---

func (d *ddlWriter) writeGrant(g *Grant) {
	target := grantTarget(g)
	if target == "" {
		return
	}
	d.P("GRANT %s ON %s TO %s;\n", strings.ToUpper(g.Privileges), target, grantTo(g.To))
}

func grantTarget(g *Grant) string {
	switch strings.ToLower(g.On) {
	case "table":
		return "TABLE " + QualifiedName(g.Schema, g.Name)
	case "all-tables-in-schema":
		return "ALL TABLES IN SCHEMA " + QuoteIdent(g.Schema)
	case "schema":
		return "SCHEMA " + QuoteIdent(g.Name)
	case "sequence":
		return "SEQUENCE " + QualifiedName(g.Schema, g.Name)
	case "all-sequences-in-schema":
		return "ALL SEQUENCES IN SCHEMA " + QuoteIdent(g.Schema)
	case "function":
		return "FUNCTION " + QualifiedName(g.Schema, g.Name)
	case "all-functions-in-schema":
		return "ALL FUNCTIONS IN SCHEMA " + QuoteIdent(g.Schema)
	case "type":
		return "TYPE " + QualifiedName(g.Schema, g.Name)
	case "database":
		return "DATABASE " + QuoteIdent(g.Name)
	}
	return ""
}

func grantTo(to string) string {
	var parts []string
	for _, role := range strings.Split(to, ",") {
		role = strings.TrimSpace(role)
		if role == "public" || role == "PUBLIC" {
			parts = append(parts, "PUBLIC")
		} else {
			parts = append(parts, QuoteIdent(role))
		}
	}
	return strings.Join(parts, ", ")
}

func (d *ddlWriter) writeGrantRole(gr *GrantRole) {
	d.P("GRANT %s TO %s", QuoteIdent(gr.Role), grantTo(gr.To))
	d.If(gr.WithInherit == "true", " WITH INHERIT TRUE")
	d.S(";\n")
}

// --- index ---

func (d *ddlWriter) writeIndex(q string, idx *Index) {
	if idx.Unique == "true" {
		d.S("CREATE UNIQUE INDEX ")
	} else {
		d.S("CREATE INDEX ")
	}
	d.P("%s ON %s%s", QuoteIdent(idx.Name), q, QuoteIdent(idx.Table))
	d.If(idx.Using != "" && idx.Using != "btree", " USING %s", strings.ToUpper(idx.Using))
	d.P(" (\n%s\n)", indexElements(idx))
	if idx.With != nil && len(idx.With.Params) > 0 {
		var params []string
		for _, p := range idx.With.Params {
			params = append(params, fmt.Sprintf("%s='%s'", p.Name, p.Value))
		}
		d.P("\n\tWITH (%s)", strings.Join(params, ", "))
	}
	if idx.Where != nil && idx.Where.Value != "" {
		d.P("\n\tWHERE %s", idx.Where.Value)
	}
	d.S(";\n\n")
}

// WriteIndex writes CREATE INDEX DDL for a single index.
// Public API used by diff engine.
func WriteIndex(w io.Writer, q string, idx *Index) error {
	d := &ddlWriter{w: w}
	d.writeIndex(q, idx)
	return d.Err()
}

func indexElements(idx *Index) string {
	var parts []string
	for _, col := range idx.Columns {
		s := "\t" + QuoteIdent(col.Name)
		if col.Opclass != "" {
			s += " " + col.Opclass
		}
		if col.Order == "desc" {
			s += " DESC"
		}
		switch col.Nulls {
		case "first":
			s += " NULLS FIRST"
		case "last":
			s += " NULLS LAST"
		}
		parts = append(parts, s)
	}
	for _, expr := range idx.Expressions {
		v := expr.Value
		if !strings.HasPrefix(v, "(") {
			v = "(" + v + ")"
		}
		parts = append(parts, "\t"+v)
	}
	return strings.Join(parts, ",\n")
}

// --- FK ---

func (d *ddlWriter) writeFK(q string, table string, fk *ForeignKey) {
	localCols := quotedNames(fk.Columns, func(c FKCol) string { return c.Name })
	refCols := quotedNames(fk.Columns, func(c FKCol) string { return c.References })

	if fk.Name != "" {
		d.P("ALTER TABLE %s%s ADD CONSTRAINT %s FOREIGN KEY (%s)\n", q, QuoteIdent(table), QuoteIdent(fk.Name), localCols)
	} else {
		d.P("ALTER TABLE %s%s ADD FOREIGN KEY (%s)\n", q, QuoteIdent(table), localCols)
	}
	d.P("\tREFERENCES %s(%s)\n", QuoteIdentQualified(fk.ToTable), refCols)
	d.P("\tON DELETE %s\n", refAction(fk.OnDelete))
	d.P("\tON UPDATE %s", refAction(fk.OnUpdate))

	if fk.Deferrable == "true" {
		d.S("\n\tDEFERRABLE")
		d.If(fk.Initially != "", " INITIALLY %s", strings.ToUpper(fk.Initially))
	} else {
		d.S("\n\tNOT DEFERRABLE")
	}

	d.S(";\n\n")
}

// WriteFK writes ALTER TABLE ADD FK DDL for a single foreign key.
// Public API used by diff engine.
func WriteFK(w io.Writer, q string, table string, fk *ForeignKey) error {
	d := &ddlWriter{w: w}
	d.writeFK(q, table, fk)
	return d.Err()
}

// GenerateDDL returns the full DDL script as a string.
func GenerateDDL(p *Project) string {
	var b strings.Builder
	_ = WriteDDL(&b, p)
	return b.String()
}

// GenerateTableDDL generates DDL for a single table: CREATE TABLE + indexes + FK.
// tableName can be qualified ("schema.table") or unqualified ("table").
func GenerateTableDDL(p *Project, tableName string) string {
	var schemaFilter, shortName string
	if dot := strings.IndexByte(tableName, '.'); dot >= 0 {
		schemaFilter = tableName[:dot]
		shortName = tableName[dot+1:]
	} else {
		shortName = tableName
	}

	var b strings.Builder
	for i := range p.Schemas {
		s := &p.Schemas[i]
		if schemaFilter != "" && s.Name != schemaFilter {
			continue
		}
		q := SchemaQualifier(s.Name)
		for j := range s.Tables {
			t := &s.Tables[j]
			if t.Name != shortName {
				continue
			}
			d := &ddlWriter{w: &b}
			d.writeTable(q, t)
			for k := range s.Indexes {
				if s.Indexes[k].Table == shortName {
					d.writeIndex(q, &s.Indexes[k])
				}
			}
			for k := range t.FKs {
				d.writeFK(q, t.Name, &t.FKs[k])
			}
			d.writeTableComments(q, t)
			d.writeColumnStorage(q, t)
			return b.String()
		}
	}
	return ""
}

// --- helpers ---

func SchemaQualifier(name string) string {
	if name == "" || name == "public" {
		return ""
	}
	return QuoteIdent(name) + "."
}

func QualifiedName(schema, name string) string {
	if schema != "" && schema != "public" {
		return QuoteIdent(schema) + "." + QuoteIdent(name)
	}
	return QuoteIdent(name)
}

func skipSet(s *Schema) map[string]bool {
	m := make(map[string]bool)
	for i := range s.Tables {
		if s.Tables[i].Generate == "false" {
			m[s.Tables[i].Name] = true
		}
	}
	return m
}

func QuoteIdent(name string) string {
	return `"` + name + `"`
}

// QuoteIdentQualified handles "schema.table" → "schema"."table".
func QuoteIdentQualified(name string) string {
	if i := strings.IndexByte(name, '.'); i >= 0 {
		return QuoteIdent(name[:i]) + "." + QuoteIdent(name[i+1:])
	}
	return QuoteIdent(name)
}

func QuotedColList(cols []ColRef) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = QuoteIdent(c.Name)
	}
	return strings.Join(parts, ", ")
}

func quotedNames(cols []FKCol, f func(FKCol) string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = QuoteIdent(f(c))
	}
	return strings.Join(parts, ", ")
}

func isSerial(t string) bool {
	switch strings.ToLower(t) {
	case "serial", "bigserial", "smallserial":
		return true
	}
	return false
}

func isNumericWithParams(c *Column) bool {
	t := strings.ToLower(c.Type)
	return (t == "numeric" || t == "decimal") && c.Precision > 0
}

func hasLength(c *Column) bool {
	if c.Length <= 0 {
		return false
	}
	switch strings.ToLower(c.Type) {
	case "varchar", "character varying", "char", "character", "bit", "varbit", "bit varying":
		return true
	}
	return false
}

func refAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "restrict":
		return "RESTRICT"
	case "cascade":
		return "CASCADE"
	case "set-null", "set null":
		return "SET NULL"
	case "set-default", "set default":
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

package diff

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	sqlconv "github.com/vmkteam/pgdesigner/pkg/format/sql"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// tstzRe matches timestamp with timezone offset in partition bounds.
var tstzRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[+-]\d{2}`)

// DiffResult holds the list of changes between two Projects.
type DiffResult struct {
	Changes []Change
}

// SQL returns the full migration script with hazard comments.
func (r *DiffResult) SQL() string {
	var b strings.Builder
	for _, c := range r.Changes {
		for _, h := range c.Hazards {
			b.WriteString(sqlComment("[%s] %s: %s", h.Level, h.Code, h.Message) + "\n")
		}
		b.WriteString(c.SQL)
		b.WriteString("\n\n")
	}
	return b.String()
}

// sqlComment returns an SQL line comment. Line breaks, which object names may contain,
// are replaced so the comment can't end early and leave the rest as SQL.
func sqlComment(format string, args ...any) string {
	return "-- " + strings.NewReplacer("\r", " ", "\n", " ").Replace(fmt.Sprintf(format, args...))
}

// HasHazards reports whether any change has hazards.
func (r *DiffResult) HasHazards() bool {
	for _, c := range r.Changes {
		if len(c.Hazards) > 0 {
			return true
		}
	}
	return false
}

// Errors returns changes with dangerous hazards.
func (r *DiffResult) Errors() []Change {
	var out []Change
	for _, c := range r.Changes {
		for _, h := range c.Hazards {
			if h.Level == "dangerous" {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// Change describes a single schema change.
type Change struct {
	Schema  string   // schema name
	Object  string   // table, column, index, fk, pk, unique, check, enum
	Action  string   // add, drop, alter
	Table   string   // parent table (for column/constraint changes)
	Name    string   // object name
	SQL     string   // generated ALTER/CREATE/DROP statement
	Hazards []Hazard // warnings
}

// Hazard is a migration risk warning.
type Hazard struct {
	Level   string // dangerous, warning, info
	Code    string // DELETES_DATA, TABLE_REWRITE, ACCESS_EXCLUSIVE, LONG_RUNNING, BACKFILL_REQUIRED
	Message string
}

// Diff compares two Projects and returns the changes needed to migrate old → updated.
func Diff(old, updated *pgd.Project) *DiffResult {
	b := &diffBuilder{recreated: make(map[string][]string)}
	oldEnums, newEnums := projectEnums(old), projectEnums(updated)

	// Phase 1: DROP in reverse dependency order, each step over all schemas: FKs before
	// the keys and tables they reference, then keys and checks, indexes, columns, tables
	b.eachOldSchema(old, updated, b.dropFKs)
	b.eachOldSchema(old, updated, b.dropKeys)
	b.eachOldSchema(old, updated, b.dropIndexes)
	b.eachOldSchema(old, updated, b.dropColumns)
	b.dropTables(old, updated)

	// Phase 2: new schemas and enums, before the tables and columns that use them
	b.createSchemas(old, updated)
	b.diffEnums(oldEnums, newEnums)

	// Phase 3: tables and columns
	b.eachSchema(old, updated, b.createTables)

	// Phase 4: indexes, PK, UNIQUE, CHECK, partitions
	b.eachSchema(old, updated, b.createConstraints)

	// Phase 5: FK, once every referenced table and key exists
	b.eachSchema(old, updated, b.createFKs)

	// Phase 6: table and column comments
	b.diffComments(old, updated)

	// Phase 7: drop removed enums, after the columns using them are dropped or altered
	b.dropEnums(oldEnums, newEnums)

	return &DiffResult{Changes: b.changes}
}

// diffBuilder accumulates changes.
type diffBuilder struct {
	schema    string
	changes   []Change
	recreated map[string][]string // "schema.table" → columns dropped and re-added
}

// eachOldSchema runs phase for each old schema and its updated version (nil for a dropped schema).
func (b *diffBuilder) eachOldSchema(old, updated *pgd.Project, phase func(old, updated *pgd.Schema)) {
	for i := range old.Schemas {
		os := &old.Schemas[i]
		b.schema = os.Name
		phase(os, findSchema(updated.Schemas, os.Name))
	}
}

// eachSchema runs phase for each updated schema and its old version (nil for a new schema).
func (b *diffBuilder) eachSchema(old, updated *pgd.Project, phase func(old, updated *pgd.Schema)) {
	for i := range updated.Schemas {
		ns := &updated.Schemas[i]
		b.schema = ns.Name
		phase(findSchema(old.Schemas, ns.Name), ns)
	}
}

func (b *diffBuilder) add(c Change) {
	if c.Schema == "" {
		c.Schema = b.schema
	}
	b.changes = append(b.changes, c)
}

func (b *diffBuilder) q(name string) string {
	return pgd.QuoteIdent(name)
}

func (b *diffBuilder) qt(table string) string {
	q := pgd.SchemaQualifier(b.schema)
	return q + pgd.QuoteIdent(table)
}

// --- Phase 1: DROP ---

// dropFKs drops removed and changed FKs of tables that remain.
func (b *diffBuilder) dropFKs(old, updated *pgd.Schema) {
	newTables := tableMap(nilTables(updated))
	for _, ot := range old.Tables {
		if nt, exists := newTables[ot.Name]; exists {
			b.dropRemovedFK(ot.Name, ot.FKs, nt.FKs)
		}
	}
}

// dropKeys drops removed and changed CHECK, UNIQUE and PK constraints of tables that remain.
func (b *diffBuilder) dropKeys(old, updated *pgd.Schema) {
	newTables := tableMap(nilTables(updated))
	for _, ot := range old.Tables {
		if nt, exists := newTables[ot.Name]; exists {
			b.dropRemovedChecks(ot.Name, ot.Checks, nt.Checks)
			b.dropRemovedUniques(ot.Name, ot.Uniques, nt.Uniques)
			b.dropModifiedPK(ot.Name, ot.PK, nt.PK)
		}
	}
}

func (b *diffBuilder) dropIndexes(old, updated *pgd.Schema) {
	newIndexes := indexMap(nilIndexes(updated))
	renamed := renamedIndexes(nilIndexes(updated), indexMap(old.Indexes))
	for i := range old.Indexes {
		oi := &old.Indexes[i]
		ni, exists := newIndexes[oi.Name]
		if !exists {
			ni = renamed.take(indexIdentity(oi)) // the index may be renamed
		}
		if ni == nil || indexChanged(oi, ni) {
			b.dropIndex(oi.Name)
		}
	}
}

func (b *diffBuilder) dropColumns(old, updated *pgd.Schema) {
	newTables := tableMap(nilTables(updated))
	for _, ot := range old.Tables {
		if nt, exists := newTables[ot.Name]; exists {
			b.dropRemovedColumns(ot.Name, ot.Columns, nt.Columns)
		}
	}
}

// droppedTable is a table of an old schema missing in the updated project.
type droppedTable struct {
	schema string
	*pgd.Table
}

func (t droppedTable) key() string { return t.schema + "." + t.Name }

// fkTarget returns the schema-qualified table referenced by an FK of a table in schema.
func fkTarget(schema string, fk *pgd.ForeignKey) string {
	if strings.Contains(fk.ToTable, ".") {
		return fk.ToTable
	}
	return schema + "." + fk.ToTable
}

// dropTables drops removed tables of all schemas so that a table goes before the tables
// its FKs reference. FKs within a reference cycle are dropped first.
func (b *diffBuilder) dropTables(old, updated *pgd.Project) {
	var tables []droppedTable
	for i := range old.Schemas {
		os := &old.Schemas[i]
		newTables := tableMap(nilTables(findSchema(updated.Schemas, os.Name)))
		for j := range os.Tables {
			if _, exists := newTables[os.Tables[j].Name]; !exists {
				tables = append(tables, droppedTable{os.Name, &os.Tables[j]})
			}
		}
	}

	for len(tables) > 0 {
		dropped := make(map[string]bool, len(tables))
		for _, t := range tables {
			dropped[t.key()] = true
		}
		var referenced []droppedTable
		for _, t := range tables {
			if isReferenced(t, tables) {
				referenced = append(referenced, t)
				continue
			}
			b.schema = t.schema
			b.dropTable(t.Name)
		}
		if len(referenced) == len(tables) { // a reference cycle
			for _, t := range tables {
				b.schema = t.schema
				for k := range t.FKs {
					if target := fkTarget(t.schema, &t.FKs[k]); target != t.key() && dropped[target] {
						b.dropFK(t.Name, &t.FKs[k])
					}
				}
			}
			for _, t := range tables {
				b.schema = t.schema
				b.dropTable(t.Name)
			}
			return
		}
		tables = referenced
	}
}

// isReferenced reports whether an FK of another table in tables references t.
func isReferenced(t droppedTable, tables []droppedTable) bool {
	for _, u := range tables {
		if u.key() == t.key() {
			continue
		}
		for k := range u.FKs {
			if fkTarget(u.schema, &u.FKs[k]) == t.key() {
				return true
			}
		}
	}
	return false
}

// --- Phases 3–5: CREATE/ALTER ---

func (b *diffBuilder) createTables(old, updated *pgd.Schema) {
	oldTables := tableMap(nilTables(old))

	// add new tables
	for i := range updated.Tables {
		if _, exists := oldTables[updated.Tables[i].Name]; !exists {
			b.addTable(&updated.Tables[i])
		}
	}

	// alter columns and options of existing tables
	for i := range updated.Tables {
		nt := &updated.Tables[i]
		if ot, exists := oldTables[nt.Name]; exists {
			b.diffColumns(nt.Name, ot.Columns, nt.Columns)
			b.diffTableOptions(ot, nt)
		}
	}
}

// diffTableOptions alters storage parameters, persistence, access method and tablespace.
func (b *diffBuilder) diffTableOptions(old, updated *pgd.Table) {
	t := b.qt(updated.Name)
	alter := func(sql string, hazards ...Hazard) {
		b.add(Change{Object: "table", Action: "alter", Table: updated.Name, Name: updated.Name, SQL: sql, Hazards: hazards})
	}
	rewrite := func(what string) Hazard {
		return Hazard{Level: "warning", Code: "TABLE_REWRITE", Message: fmt.Sprintf("%s rewrites table %s", what, updated.Name)}
	}

	oldParams, newParams := withMap(old.With), withMap(updated.With)
	var set []pgd.WithParam
	var reset []string
	if updated.With != nil {
		for _, p := range updated.With.Params {
			if v, exists := oldParams[p.Name]; !exists || v != p.Value {
				set = append(set, p)
			}
		}
	}
	if old.With != nil {
		for _, p := range old.With.Params {
			if _, exists := newParams[p.Name]; !exists {
				reset = append(reset, p.Name)
			}
		}
	}
	if len(set) > 0 {
		alter(fmt.Sprintf("ALTER TABLE %s SET (%s);", t, pgd.WithParams(set)))
	}
	if len(reset) > 0 {
		alter(fmt.Sprintf("ALTER TABLE %s RESET (%s);", t, strings.Join(reset, ", ")))
	}

	if (old.Unlogged == "true") != (updated.Unlogged == "true") {
		persistence := "LOGGED"
		if updated.Unlogged == "true" {
			persistence = "UNLOGGED"
		}
		alter(fmt.Sprintf("ALTER TABLE %s SET %s;", t, persistence), rewrite("SET "+persistence))
	}
	if cmp.Or(old.Using, "heap") != cmp.Or(updated.Using, "heap") {
		alter(fmt.Sprintf("ALTER TABLE %s SET ACCESS METHOD %s;", t, b.q(cmp.Or(updated.Using, "heap"))), rewrite("SET ACCESS METHOD"))
	}
	if old.Tablespace != updated.Tablespace {
		alter(fmt.Sprintf("ALTER TABLE %s SET TABLESPACE %s;", t, b.q(cmp.Or(updated.Tablespace, "pg_default"))), rewrite("SET TABLESPACE"))
	}
}

func withMap(w *pgd.With) map[string]string {
	m := make(map[string]string)
	if w != nil {
		for _, p := range w.Params {
			m[p.Name] = p.Value
		}
	}
	return m
}

func (b *diffBuilder) createConstraints(old, updated *pgd.Schema) {
	oldTables := tableMap(nilTables(old))

	// add/recreate indexes
	oldIndexes := indexMap(nilIndexes(old))
	renamed := renamedIndexes(nilIndexes(old), indexMap(updated.Indexes))
	for i := range updated.Indexes {
		ni := &updated.Indexes[i]
		oi, exists := oldIndexes[ni.Name]
		if !exists {
			oi = renamed.take(indexIdentity(ni)) // the index may be renamed
		}
		switch {
		case oi == nil || indexChanged(oi, ni) || b.indexRecreated(ni):
			b.addIndex(ni)
		case oi.Name != ni.Name:
			b.add(Change{
				Object: "index", Action: "alter", Name: ni.Name,
				SQL: fmt.Sprintf("ALTER INDEX %s RENAME TO %s;", b.qt(oi.Name), b.q(ni.Name)),
			})
		}
	}

	// add constraints on existing tables
	for _, nt := range updated.Tables {
		ot, exists := oldTables[nt.Name]
		if !exists {
			continue
		}
		b.addNewPK(nt.Name, ot.PK, nt.PK)
		b.addNewUniques(nt.Name, ot.Uniques, nt.Uniques)
		b.addNewChecks(nt.Name, ot.Checks, nt.Checks)
		b.diffPartitions(nt.Name, ot.Partitions, nt.Partitions)
		b.diffPartitionBy(nt.Name, ot.PartitionBy, nt.PartitionBy)
	}
}

func (b *diffBuilder) createFKs(old, updated *pgd.Schema) {
	oldTables := tableMap(nilTables(old))
	for _, nt := range updated.Tables {
		ot, exists := oldTables[nt.Name]
		switch {
		case exists:
			b.addNewFK(nt.Name, ot.FKs, nt.FKs)
		case nt.PartitionOf == "": // FKs of a partition come from its parent
			for i := range nt.FKs {
				b.addFK(nt.Name, &nt.FKs[i])
			}
		}
	}
}

// --- Tables ---

func (b *diffBuilder) addTable(t *pgd.Table) {
	var sb strings.Builder
	_ = pgd.WriteTable(&sb, pgd.SchemaQualifier(b.schema), t)
	b.add(Change{
		Object: "table", Action: "add", Name: t.Name,
		SQL: strings.TrimSpace(sb.String()),
	})
	for i := range t.Columns {
		b.addColumnStorage(t.Name, &t.Columns[i])
	}
}

func (b *diffBuilder) dropTable(name string) {
	b.add(Change{
		Object: "table", Action: "drop", Name: name,
		SQL:     fmt.Sprintf("DROP TABLE %s;", b.qt(name)),
		Hazards: []Hazard{{Level: "dangerous", Code: "DELETES_DATA", Message: fmt.Sprintf("drops table %s and all its data", name)}},
	})
}

// --- Columns ---

func (b *diffBuilder) diffColumns(table string, old, updated []pgd.Column) {
	oldCols := columnMap(old)

	for _, nc := range updated {
		oc, exists := oldCols[nc.Name]
		if !exists {
			b.addColumn(table, &nc)
			continue
		}
		b.diffColumn(table, oc, &nc)
	}
}

func (b *diffBuilder) dropRemovedColumns(table string, old, updated []pgd.Column) {
	newCols := columnMap(updated)
	for _, oc := range old {
		if _, exists := newCols[oc.Name]; !exists {
			b.dropColumn(table, oc.Name)
		}
	}
}

func (b *diffBuilder) addColumn(table string, c *pgd.Column) {
	def := pgd.ColumnDef(c)
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", b.qt(table), def)

	var hazards []Hazard
	if needsBackfill(c) {
		hazards = append(hazards, backfillHazard(c))
	}

	b.add(Change{
		Object: "column", Action: "add", Table: table, Name: c.Name,
		SQL: sql, Hazards: hazards,
	})
	b.addColumnStorage(table, c)
}

// addColumnStorage emits SET STORAGE for a new column: ColumnDef doesn't include it.
func (b *diffBuilder) addColumnStorage(table string, c *pgd.Column) {
	b.diffColumnStorage(table, b.qt(table), b.q(c.Name), &pgd.Column{}, c)
}

// needsBackfill reports whether adding the column fails on a non-empty table.
func needsBackfill(c *pgd.Column) bool {
	return c.Nullable == "false" && c.Default == "" && c.Identity == nil && c.Generated == nil
}

func backfillHazard(c *pgd.Column) Hazard {
	return Hazard{
		Level: "warning", Code: "BACKFILL_REQUIRED",
		Message: fmt.Sprintf("column %s is NOT NULL without DEFAULT", c.Name),
	}
}

func (b *diffBuilder) dropColumn(table, col string) {
	b.add(Change{
		Object: "column", Action: "drop", Table: table, Name: col,
		SQL:     fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", b.qt(table), b.q(col)),
		Hazards: []Hazard{{Level: "dangerous", Code: "DELETES_DATA", Message: fmt.Sprintf("drops column %s.%s and all its data", table, col)}},
	})
}

func (b *diffBuilder) diffColumn(table string, old, updated *pgd.Column) {
	if needsRecreate(old, updated) {
		b.recreateColumn(table, updated)
		return
	}

	t := b.qt(table)
	col := b.q(updated.Name)

	b.diffColumnType(table, t, col, old, updated)
	b.diffColumnGenerated(table, t, col, old, updated) // before DEFAULT: a generated column can't have one
	b.diffColumnNullable(table, t, col, old, updated)
	b.diffColumnDefault(table, t, col, old, updated)
	b.diffColumnIdentity(table, t, col, old, updated)
	b.diffColumnCompression(table, t, col, old, updated)
	b.diffColumnStorage(table, t, col, old, updated)
}

// needsRecreate reports whether the column can't be altered in place: PostgreSQL can't
// make a regular column generated, switch STORED ↔ VIRTUAL or drop a virtual expression.
func needsRecreate(old, updated *pgd.Column) bool {
	og, ng := old.Generated, updated.Generated
	switch {
	case og == nil && ng == nil:
		return false
	case og == nil:
		return true
	case ng == nil:
		return isVirtual(og)
	default:
		return isVirtual(og) != isVirtual(ng)
	}
}

func isVirtual(g *pgd.Generated) bool {
	return g.Stored == "false"
}

// recreateColumn drops and re-adds the column. Indexes and constraints involving it are
// dropped with it and re-created in createConstraints and createFKs.
func (b *diffBuilder) recreateColumn(table string, c *pgd.Column) {
	t := b.qt(table)
	key := b.schema + "." + table
	b.recreated[key] = append(b.recreated[key], c.Name)

	hazards := []Hazard{{
		Level: "dangerous", Code: "COLUMN_RECREATE",
		Message: fmt.Sprintf("%s.%s is dropped and re-added: current values are replaced; "+
			"dependent views and FKs from other tables block DROP", table, c.Name),
	}}
	if needsBackfill(c) {
		hazards = append(hazards, backfillHazard(c))
	}
	b.add(Change{
		Object: "column", Action: "alter", Table: table, Name: c.Name,
		SQL:     fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;\nALTER TABLE %s ADD COLUMN %s;", t, b.q(c.Name), t, pgd.ColumnDef(c)),
		Hazards: hazards,
	})
	b.addColumnStorage(table, c)
}

// diffColumnGenerated alters a generated column in place; needsRecreate covers the rest.
func (b *diffBuilder) diffColumnGenerated(table, t, col string, old, updated *pgd.Column) {
	og, ng := old.Generated, updated.Generated
	switch {
	case og != nil && ng == nil: // stored → regular, values are kept
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP EXPRESSION;", t, col),
		})
	case og != nil && ng != nil && !sameExpr(og.Expression, ng.Expression):
		var hazards []Hazard
		if !isVirtual(ng) {
			hazards = append(hazards, Hazard{
				Level: "warning", Code: "TABLE_REWRITE",
				Message: fmt.Sprintf("new expression of stored column %s.%s rewrites the table", table, updated.Name),
			})
		}
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL:     fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET EXPRESSION AS (%s);", t, col, ng.Expression),
			Hazards: hazards,
		})
	}
}

func (b *diffBuilder) diffColumnType(table, t, col string, old, updated *pgd.Column) {
	oldType := pgd.TypeSpec(old)
	newType := pgd.TypeSpec(updated)
	if oldType == newType {
		return
	}
	var hazards []Hazard
	if !isCompatibleCast(oldType, newType) {
		hazards = append(hazards, Hazard{
			Level: "warning", Code: "TABLE_REWRITE",
			Message: fmt.Sprintf("changing %s.%s from %s to %s may rewrite table", table, updated.Name, oldType, newType),
		})
	}
	b.add(Change{
		Object: "column", Action: "alter", Table: table, Name: updated.Name,
		SQL:     fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", t, col, newType),
		Hazards: hazards,
	})
}

func (b *diffBuilder) diffColumnNullable(table, t, col string, old, updated *pgd.Column) {
	if old.Nullable == updated.Nullable {
		return
	}
	if updated.Nullable == "false" {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;", t, col),
		})
	} else {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;", t, col),
		})
	}
}

func (b *diffBuilder) diffColumnDefault(table, t, col string, old, updated *pgd.Column) {
	if old.Default == updated.Default || (old.Default != "" && updated.Default != "" && sameExpr(old.Default, updated.Default)) {
		return
	}
	if updated.Default == "" {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;", t, col),
		})
	} else {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;", t, col, updated.Default),
		})
	}
}

func (b *diffBuilder) diffColumnIdentity(table, t, col string, old, updated *pgd.Column) {
	oldID := identityStr(old.Identity)
	newID := identityStr(updated.Identity)
	if oldID == newID {
		return
	}

	b.diffIdentityMode(table, t, col, old, updated)
	b.diffIdentitySeqOpts(table, t, col, old, updated)
}

func (b *diffBuilder) diffIdentityMode(table, t, col string, old, updated *pgd.Column) {
	switch {
	case old.Identity == nil && updated.Identity != nil:
		gen := identityGenStr(updated.Identity.Generated)
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s ADD GENERATED %s AS IDENTITY;", t, col, gen),
		})
	case old.Identity != nil && updated.Identity == nil:
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP IDENTITY;", t, col),
		})
	case old.Identity != nil && updated.Identity != nil && old.Identity.Generated != updated.Identity.Generated:
		gen := identityGenStr(updated.Identity.Generated)
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET GENERATED %s;", t, col, gen),
		})
	}
}

func identityGenStr(gen string) string {
	if gen == "always" {
		return "ALWAYS"
	}
	return "BY DEFAULT"
}

func (b *diffBuilder) diffIdentitySeqOpts(table, t, col string, old, updated *pgd.Column) {
	if old.Identity == nil || updated.Identity == nil || old.Identity.Generated != updated.Identity.Generated {
		return
	}
	oldSeq := seqOptStr(old.Identity.Sequence)
	newSeq := seqOptStr(updated.Identity.Sequence)
	if oldSeq == newSeq || updated.Identity.Sequence == nil {
		return
	}
	seq := updated.Identity.Sequence
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
	} else {
		opts = append(opts, "NO CYCLE")
	}
	if len(opts) > 0 {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET %s;", t, col, strings.Join(opts, " ")),
		})
	}
}

func (b *diffBuilder) diffColumnCompression(table, t, col string, old, updated *pgd.Column) {
	if old.Compression == updated.Compression {
		return
	}
	if updated.Compression == "" {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET COMPRESSION DEFAULT;", t, col),
		})
	} else {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET COMPRESSION %s;", t, col, updated.Compression),
		})
	}
}

func (b *diffBuilder) diffColumnStorage(table, t, col string, old, updated *pgd.Column) {
	if old.Storage == updated.Storage || updated.Storage == "" {
		return
	}
	b.add(Change{
		Object: "column", Action: "alter", Table: table, Name: updated.Name,
		SQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET STORAGE %s;", t, col, strings.ToUpper(updated.Storage)),
	})
}

// --- Partitions ---

func (b *diffBuilder) diffPartitionBy(table string, old, updated *pgd.PartitionBy) {
	oldStr := partitionByStr(old)
	newStr := partitionByStr(updated)
	if oldStr == newStr {
		return
	}
	t := b.qt(table)
	switch {
	case old == nil && updated != nil:
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL:     sqlComment("Cannot add PARTITION BY to existing table %s; requires table recreation", t),
			Hazards: []Hazard{{Level: "dangerous", Code: "TABLE_RECREATE", Message: "adding PARTITION BY requires table recreation"}},
		})
	case old != nil && updated == nil:
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL:     sqlComment("Cannot remove PARTITION BY from table %s; requires table recreation", t),
			Hazards: []Hazard{{Level: "dangerous", Code: "TABLE_RECREATE", Message: "removing PARTITION BY requires table recreation"}},
		})
	default:
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL:     sqlComment("Cannot change PARTITION BY strategy on table %s; requires table recreation", t),
			Hazards: []Hazard{{Level: "dangerous", Code: "TABLE_RECREATE", Message: "changing PARTITION BY strategy requires table recreation"}},
		})
	}
}

func partitionByStr(pb *pgd.PartitionBy) string {
	if pb == nil {
		return ""
	}
	return pb.Type + ":" + pgd.QuotedColList(pb.Columns)
}

func (b *diffBuilder) diffPartitions(table string, old, updated []pgd.Partition) {
	oldMap := partitionMap(old)
	newMap := partitionMap(updated)
	t := b.qt(table)

	// Detach removed partitions
	for _, op := range old {
		if _, exists := newMap[op.Name]; !exists {
			b.add(Change{
				Object: "partition", Action: "drop", Table: table, Name: op.Name,
				SQL:     fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s;", t, b.qt(op.Name)),
				Hazards: []Hazard{{Level: "warning", Code: "DETACH_PARTITION", Message: fmt.Sprintf("detaches partition %s from %s", op.Name, table)}},
			})
		}
	}

	// Attach new partitions
	for _, np := range updated {
		op, exists := oldMap[np.Name]
		if !exists {
			b.add(Change{
				Object: "partition", Action: "add", Table: table, Name: np.Name,
				SQL: fmt.Sprintf("CREATE TABLE %s PARTITION OF %s\n    %s;", b.qt(np.Name), t, np.Bound),
			})
			continue
		}
		// Bound changed → detach + attach (normalize timestamps for TZ-aware comparison)
		if normalizeBound(op.Bound) != normalizeBound(np.Bound) {
			b.add(Change{
				Object: "partition", Action: "alter", Table: table, Name: np.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s;\nALTER TABLE %s ATTACH PARTITION %s %s;",
					t, b.qt(np.Name), t, b.qt(np.Name), np.Bound),
				Hazards: []Hazard{{Level: "warning", Code: "REATTACH_PARTITION", Message: fmt.Sprintf("detach+attach partition %s with new bounds", np.Name)}},
			})
		}
	}
}

// normalizeBound normalizes partition bound strings by converting timezone-aware
// timestamps to UTC, so that '2022-01-01 00:00:00+00' == '2022-01-01 03:00:00+03'.
func normalizeBound(s string) string {
	return tstzRe.ReplaceAllStringFunc(s, func(m string) string {
		t, err := time.Parse("2006-01-02 15:04:05-07", m)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05+07", m)
		}
		if err != nil {
			return m
		}
		return t.UTC().Format("2006-01-02 15:04:05+00")
	})
}

func partitionMap(parts []pgd.Partition) map[string]*pgd.Partition {
	m := make(map[string]*pgd.Partition, len(parts))
	for i := range parts {
		m[parts[i].Name] = &parts[i]
	}
	return m
}

// --- Indexes ---

func (b *diffBuilder) addIndex(idx *pgd.Index) {
	var sb strings.Builder
	_ = pgd.WriteIndex(&sb, pgd.SchemaQualifier(b.schema), idx)
	b.add(Change{
		Object: "index", Action: "add", Name: idx.Name,
		SQL: strings.TrimSpace(sb.String()),
	})
}

func (b *diffBuilder) dropIndex(name string) {
	b.add(Change{
		Object: "index", Action: "drop", Name: name,
		SQL: fmt.Sprintf("DROP INDEX %s;", b.qt(name)),
	})
}

// --- FK ---

func (b *diffBuilder) dropRemovedFK(table string, old, updated []pgd.ForeignKey) {
	newFKs := fkMap(updated)
	renamed := renamedFKs(updated, fkMap(old))
	for i := range old {
		ofk := &old[i]
		nfk, exists := newFKs[fkKey(ofk)]
		if !exists {
			nfk = renamed.take(fkIdentity(ofk)) // the FK may be renamed
		}
		if nfk == nil || fkChanged(ofk, nfk) {
			b.dropFK(table, ofk)
		}
	}
}

func (b *diffBuilder) addNewFK(table string, old, updated []pgd.ForeignKey) {
	oldFKs := fkMap(old)
	renamed := renamedFKs(old, fkMap(updated))
	for i := range updated {
		nfk := &updated[i]
		ofk, exists := oldFKs[fkKey(nfk)]
		if !exists {
			ofk = renamed.take(fkIdentity(nfk)) // the FK may be renamed
		}
		switch {
		case ofk == nil || fkChanged(ofk, nfk) || b.touchesRecreated(table, fkColNames(nfk.Columns)):
			b.addFK(table, nfk)
		case ofk.Name != nfk.Name && ofk.Name != "" && nfk.Name != "":
			b.add(Change{
				Object: "fk", Action: "alter", Table: table, Name: nfk.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s;", b.qt(table), b.q(ofk.Name), b.q(nfk.Name)),
			})
		}
	}
}

func (b *diffBuilder) addFK(table string, fk *pgd.ForeignKey) {
	var sb strings.Builder
	_ = pgd.WriteFK(&sb, pgd.SchemaQualifier(b.schema), table, fk)
	b.add(Change{
		Object: "fk", Action: "add", Table: table, Name: fk.Name,
		SQL: strings.TrimSpace(sb.String()),
	})
}

func (b *diffBuilder) dropFK(table string, fk *pgd.ForeignKey) {
	cols := strings.Join(fkColNames(fk.Columns), ", ")
	b.dropConstraint("fk", table, fk.Name, fmt.Sprintf("FOREIGN KEY (%s) to %s", cols, fk.ToTable))
}

// dropConstraint drops a constraint by name. An unnamed one (inline constraints parsed
// from SQL have no name) can't be addressed: it is reported instead.
func (b *diffBuilder) dropConstraint(object, table, name, def string) {
	if name == "" {
		b.add(Change{
			Object: object, Action: "drop", Table: table,
			SQL:     sqlComment("Cannot drop unnamed %s on %s: name it in the old model", def, b.qt(table)),
			Hazards: []Hazard{{Level: "warning", Code: "UNNAMED_CONSTRAINT", Message: fmt.Sprintf("unnamed %s on %s is not dropped", def, table)}},
		})
		return
	}
	b.add(Change{
		Object: object, Action: "drop", Table: table, Name: name,
		SQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.qt(table), b.q(name)),
	})
}

// --- PK ---

func (b *diffBuilder) dropModifiedPK(table string, old, updated *pgd.PrimaryKey) {
	if old == nil {
		return
	}
	if updated == nil || pkChanged(old, updated) {
		b.dropConstraint("pk", table, cmp.Or(old.Name, table+"_pkey"), "PRIMARY KEY")
	}
}

func (b *diffBuilder) addNewPK(table string, old, updated *pgd.PrimaryKey) {
	if updated == nil {
		return
	}
	if old == nil || pkChanged(old, updated) || b.touchesRecreated(table, pgd.ColRefNames(updated.Columns)) {
		cols := pgd.QuotedColList(updated.Columns)
		name := updated.Name
		if name == "" {
			name = table + "_pkey"
		}
		b.add(Change{
			Object: "pk", Action: "add", Table: table, Name: name,
			SQL: fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s);", b.qt(table), b.q(name), cols),
		})
	}
}

// --- UNIQUE ---

func (b *diffBuilder) dropRemovedUniques(table string, old, updated []pgd.Unique) {
	newU := uniqueMap(updated)
	for _, ou := range old {
		if nu, exists := newU[uniqueKey(&ou)]; !exists || uniqueChanged(&ou, nu) {
			b.dropConstraint("unique", table, ou.Name, "UNIQUE ("+pgd.QuotedColList(ou.Columns)+")")
		}
	}
}

func (b *diffBuilder) addNewUniques(table string, old, updated []pgd.Unique) {
	oldU := uniqueMap(old)
	for _, nu := range updated {
		if ou, exists := oldU[uniqueKey(&nu)]; !exists || uniqueChanged(ou, &nu) || b.touchesRecreated(table, pgd.ColRefNames(nu.Columns)) {
			b.add(Change{
				Object: "unique", Action: "add", Table: table, Name: nu.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s ADD %s;", b.qt(table), pgd.UniqueDef(&nu)),
			})
		}
	}
}

func uniqueChanged(old, updated *pgd.Unique) bool {
	return !slices.Equal(pgd.ColRefNames(old.Columns), pgd.ColRefNames(updated.Columns)) ||
		(old.NullsDistinct == "false") != (updated.NullsDistinct == "false")
}

// --- CHECK ---

func (b *diffBuilder) dropRemovedChecks(table string, old, updated []pgd.Check) {
	newC := checkMap(updated)
	for _, oc := range old {
		if nc, exists := newC[checkKey(&oc)]; !exists || checkChanged(&oc, nc) {
			b.dropConstraint("check", table, oc.Name, "CHECK ("+oc.Expression+")")
		}
	}
}

func (b *diffBuilder) addNewChecks(table string, old, updated []pgd.Check) {
	oldC := checkMap(old)
	for _, nc := range updated {
		if oc, exists := oldC[checkKey(&nc)]; !exists || checkChanged(oc, &nc) || b.touchesRecreated(table, nil, nc.Expression) {
			constraint := ""
			if nc.Name != "" {
				constraint = "CONSTRAINT " + b.q(nc.Name) + " "
			}
			b.add(Change{
				Object: "check", Action: "add", Table: table, Name: nc.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s ADD %sCHECK (%s);", b.qt(table), constraint, nc.Expression),
			})
		}
	}
}

func checkChanged(old, updated *pgd.Check) bool {
	return !sameExpr(old.Expression, updated.Expression)
}

// sameExpr reports whether two SQL expressions are the same, ignoring formatting: a model
// from the database (pg_get_expr) and one from DDL differ in parentheses, spacing and casts.
func sameExpr(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	return a == b || sqlconv.ExprKey(a) == sqlconv.ExprKey(b)
}

// createSchemas creates the non-public schemas missing in old.
func (b *diffBuilder) createSchemas(old, updated *pgd.Project) {
	existing := pgd.NonPublicSchemas(old)
	for _, name := range pgd.NonPublicSchemas(updated) {
		if !slices.Contains(existing, name) {
			b.add(Change{Schema: name, Object: "schema", Action: "add", Name: name,
				SQL: fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", b.q(name))})
		}
	}
}

// --- Enums ---

func projectEnums(p *pgd.Project) []pgd.Enum {
	if p.Types == nil {
		return nil
	}
	return p.Types.Enums
}

func enumKey(e *pgd.Enum) string {
	return cmp.Or(e.Schema, "public") + "." + e.Name
}

func enumMap(enums []pgd.Enum) map[string]*pgd.Enum {
	m := make(map[string]*pgd.Enum, len(enums))
	for i := range enums {
		m[enumKey(&enums[i])] = &enums[i]
	}
	return m
}

func (b *diffBuilder) diffEnums(old, updated []pgd.Enum) {
	oldMap := enumMap(old)
	for i := range updated {
		ne := &updated[i]
		if oe, exists := oldMap[enumKey(ne)]; exists {
			b.diffEnumLabels(oe, ne)
			continue
		}
		var sb strings.Builder
		_ = pgd.WriteEnum(&sb, ne)
		b.addEnum(ne, "add", strings.TrimSpace(sb.String()))
	}
}

// diffEnumLabels adds new labels at their positions. PostgreSQL can't drop or reorder
// enum labels, so such changes are reported as hazards.
func (b *diffBuilder) diffEnumLabels(old, updated *pgd.Enum) {
	name := pgd.QualifiedName(updated.Schema, updated.Name)
	unsupported := func(what string) {
		b.addEnum(updated, "alter", sqlComment("Cannot %s of enum %s; requires type recreation", what, name),
			Hazard{Level: "dangerous", Code: "TYPE_RECREATE", Message: what + " requires type recreation"})
	}

	oldSet, newSet := labelSet(old.Labels), labelSet(updated.Labels)
	for _, l := range old.Labels {
		if !newSet[l] {
			unsupported("drop value " + pgd.QuoteLiteral(l))
		}
	}
	if !slices.Equal(keepLabels(old.Labels, newSet), keepLabels(updated.Labels, oldSet)) {
		unsupported("reorder values")
	}

	for i, l := range updated.Labels {
		if oldSet[l] {
			continue
		}
		pos := ""
		switch next := slices.IndexFunc(updated.Labels[i+1:], func(l string) bool { return oldSet[l] }); {
		case next < 0: // appended after all existing labels
		case i > 0:
			pos = " AFTER " + pgd.QuoteLiteral(updated.Labels[i-1])
		default:
			pos = " BEFORE " + pgd.QuoteLiteral(updated.Labels[i+1+next])
		}
		b.addEnum(updated, "alter", fmt.Sprintf("ALTER TYPE %s ADD VALUE %s%s;", name, pgd.QuoteLiteral(l), pos),
			Hazard{Level: "warning", Code: "ENUM_ADD_VALUE", Message: fmt.Sprintf(
				"new value %s of %s can't be used until the transaction commits: run the migration without a transaction or split it", pgd.QuoteLiteral(l), updated.Name)})
	}
}

func (b *diffBuilder) dropEnums(old, updated []pgd.Enum) {
	newMap := enumMap(updated)
	for i := range old {
		oe := &old[i]
		if _, exists := newMap[enumKey(oe)]; !exists {
			b.addEnum(oe, "drop", fmt.Sprintf("DROP TYPE %s;", pgd.QualifiedName(oe.Schema, oe.Name)))
		}
	}
}

func (b *diffBuilder) addEnum(e *pgd.Enum, action, sql string, hazards ...Hazard) {
	b.add(Change{Schema: cmp.Or(e.Schema, "public"), Object: "enum", Action: action, Name: e.Name, SQL: sql, Hazards: hazards})
}

func labelSet(labels []string) map[string]bool {
	m := make(map[string]bool, len(labels))
	for _, l := range labels {
		m[l] = true
	}
	return m
}

// keepLabels returns labels that are in set, in their order.
func keepLabels(labels []string, set map[string]bool) []string {
	var out []string
	for _, l := range labels {
		if set[l] {
			out = append(out, l)
		}
	}
	return out
}

// --- Comments ---

// commentKey identifies a table (column == "") or column comment.
type commentKey struct {
	schema, table, column string
}

// projectComments returns effective table and column comments, keys in model order.
// As in generated DDL, a comment from the <comments> section (the SQL converter puts
// COMMENT ON there) overrides the comment attribute.
func projectComments(p *pgd.Project) ([]commentKey, map[commentKey]string) {
	var keys []commentKey
	m := make(map[commentKey]string)
	set := func(k commentKey, v string) {
		if _, seen := m[k]; !seen {
			keys = append(keys, k)
		}
		m[k] = v
	}
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			if t.Comment != "" {
				set(commentKey{s.Name, t.Name, ""}, t.Comment)
			}
			for _, c := range t.Columns {
				if c.Comment != "" {
					set(commentKey{s.Name, t.Name, c.Name}, c.Comment)
				}
			}
		}
	}
	for _, c := range p.Comments {
		switch strings.ToLower(c.On) {
		case "table":
			set(commentKey{cmp.Or(c.Schema, "public"), c.Name, ""}, c.Value)
		case "column":
			set(commentKey{cmp.Or(c.Schema, "public"), c.Table, c.Name}, c.Value)
		}
	}
	return keys, m
}

func (b *diffBuilder) diffComments(old, updated *pgd.Project) {
	oldKeys, oldC := projectComments(old)
	newKeys, newC := projectComments(updated)

	for _, k := range newKeys {
		if v := newC[k]; v != oldC[k] || b.commentRecreated(k) {
			b.addComment(k, v)
		}
	}
	oldTargets, newTargets := commentTargets(old), commentTargets(updated)
	for _, k := range oldKeys {
		if _, exists := newC[k]; exists || oldC[k] == "" || b.commentRecreated(k) {
			continue
		}
		// comments of dropped objects are gone with them
		if oldTargets[k] && newTargets[k] {
			b.addComment(k, "")
		}
	}
}

// commentRecreated reports whether the commented column was dropped and re-added.
func (b *diffBuilder) commentRecreated(k commentKey) bool {
	return k.column != "" && slices.Contains(b.recreated[k.schema+"."+k.table], k.column)
}

// addComment emits COMMENT ON TABLE or COLUMN; an empty comment drops it.
func (b *diffBuilder) addComment(k commentKey, comment string) {
	value := "NULL"
	if comment != "" {
		value = pgd.QuoteLiteral(comment)
	}
	c := Change{Schema: k.schema, Object: "table", Action: "alter", Table: k.table, Name: k.table}
	target := pgd.Comment{On: "table", Schema: k.schema, Name: k.table}
	if k.column != "" {
		c.Object, c.Name = "column", k.column
		target = pgd.Comment{On: "column", Schema: k.schema, Table: k.table, Name: k.column}
	}
	c.SQL = fmt.Sprintf("COMMENT ON %s IS %s;", pgd.CommentTarget(&target), value)
	b.add(c)
}

// commentTargets returns the tables and columns of the project as comment keys.
func commentTargets(p *pgd.Project) map[commentKey]bool {
	m := make(map[commentKey]bool)
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			m[commentKey{s.Name, t.Name, ""}] = true
			for _, c := range t.Columns {
				m[commentKey{s.Name, t.Name, c.Name}] = true
			}
		}
	}
	return m
}

// --- Recreated columns ---

// touchesRecreated reports whether any of cols, or a column mentioned in exprs, of the
// table was dropped and re-added: objects involving it were dropped with it.
func (b *diffBuilder) touchesRecreated(table string, cols []string, exprs ...string) bool {
	for _, rc := range b.recreated[b.schema+"."+table] {
		if slices.Contains(cols, rc) {
			return true
		}
		for _, e := range exprs {
			if mentionsColumn(e, rc) {
				return true
			}
		}
	}
	return false
}

func (b *diffBuilder) indexRecreated(idx *pgd.Index) bool {
	cols := idx.KeyColumns()
	if idx.Include != nil {
		cols = append(cols, pgd.ColRefNames(idx.Include.Columns)...)
	}
	var exprs []string
	for _, k := range idx.Keys {
		if k.IsExpression() {
			exprs = append(exprs, k.Expression)
		}
	}
	if idx.Where != nil {
		exprs = append(exprs, idx.Where.Value)
	}
	return b.touchesRecreated(idx.Table, cols, exprs...)
}

// stringLiteralRe matches SQL string literals, including doubled quotes inside.
var stringLiteralRe = regexp.MustCompile(`'(?:[^']|'')*'`)

// mentionsColumn reports whether the SQL expression references column, plain or quoted,
// outside string literals and not as a function name.
func mentionsColumn(expr, column string) bool {
	re := regexp.MustCompile(`(^|[^\w$"])("` + regexp.QuoteMeta(column) + `"|` + regexp.QuoteMeta(column) + `)($|[^\w$"(])`)
	return re.MatchString(stringLiteralRe.ReplaceAllString(expr, "''"))
}

// --- helpers ---

func findSchema(schemas []pgd.Schema, name string) *pgd.Schema {
	for i := range schemas {
		if schemas[i].Name == name {
			return &schemas[i]
		}
	}
	return nil
}

func tableMap(tables []pgd.Table) map[string]*pgd.Table {
	m := make(map[string]*pgd.Table, len(tables))
	for i := range tables {
		m[tables[i].Name] = &tables[i]
	}
	return m
}

func columnMap(cols []pgd.Column) map[string]*pgd.Column {
	m := make(map[string]*pgd.Column, len(cols))
	for i := range cols {
		m[cols[i].Name] = &cols[i]
	}
	return m
}

// renameCandidates are objects whose names are absent in the other model, grouped by
// identity in model order: a rename pairs them one-to-one, first with first.
type renameCandidates[T any] map[string][]*T

// take returns the next candidate with the identity, or nil.
func (c renameCandidates[T]) take(id string) *T {
	q := c[id]
	if len(q) == 0 {
		return nil
	}
	c[id] = q[1:]
	return q[0]
}

// renamedIndexes returns indexes whose names are absent in other: rename candidates.
func renamedIndexes(indexes []pgd.Index, other map[string]*pgd.Index) renameCandidates[pgd.Index] {
	c := renameCandidates[pgd.Index]{}
	for i := range indexes {
		if _, exists := other[indexes[i].Name]; !exists {
			id := indexIdentity(&indexes[i])
			c[id] = append(c[id], &indexes[i])
		}
	}
	return c
}

func indexMap(indexes []pgd.Index) map[string]*pgd.Index {
	m := make(map[string]*pgd.Index, len(indexes))
	for i := range indexes {
		m[indexes[i].Name] = &indexes[i]
	}
	return m
}

// renamedFKs returns FKs whose keys are absent in other: rename candidates.
func renamedFKs(fks []pgd.ForeignKey, other map[string]*pgd.ForeignKey) renameCandidates[pgd.ForeignKey] {
	c := renameCandidates[pgd.ForeignKey]{}
	for i := range fks {
		if _, exists := other[fkKey(&fks[i])]; !exists {
			id := fkIdentity(&fks[i])
			c[id] = append(c[id], &fks[i])
		}
	}
	return c
}

// fkIdentity identifies an FK by its target table and column mapping.
func fkIdentity(fk *pgd.ForeignKey) string {
	var b strings.Builder
	b.WriteString(fk.ToTable)
	for _, c := range fk.Columns {
		b.WriteString("|" + c.Name + ">" + c.References)
	}
	return b.String()
}

// Constraints are keyed by name. Inline constraints parsed from SQL have no name,
// so an unnamed constraint is keyed by its definition.

func fkKey(fk *pgd.ForeignKey) string {
	return cmp.Or(fk.Name, "\x00"+fkIdentity(fk))
}

func uniqueKey(u *pgd.Unique) string {
	return cmp.Or(u.Name, "\x00"+strings.Join(pgd.ColRefNames(u.Columns), ",")+"|"+strconv.FormatBool(u.NullsDistinct == "false"))
}

func checkKey(c *pgd.Check) string {
	return cmp.Or(c.Name, "\x00"+strings.TrimSpace(c.Expression))
}

func fkMap(fks []pgd.ForeignKey) map[string]*pgd.ForeignKey {
	m := make(map[string]*pgd.ForeignKey, len(fks))
	for i := range fks {
		m[fkKey(&fks[i])] = &fks[i]
	}
	return m
}

func uniqueMap(us []pgd.Unique) map[string]*pgd.Unique {
	m := make(map[string]*pgd.Unique, len(us))
	for i := range us {
		m[uniqueKey(&us[i])] = &us[i]
	}
	return m
}

func checkMap(cs []pgd.Check) map[string]*pgd.Check {
	m := make(map[string]*pgd.Check, len(cs))
	for i := range cs {
		m[checkKey(&cs[i])] = &cs[i]
	}
	return m
}

func nilTables(s *pgd.Schema) []pgd.Table {
	if s == nil {
		return nil
	}
	return s.Tables
}

func nilIndexes(s *pgd.Schema) []pgd.Index {
	if s == nil {
		return nil
	}
	return s.Indexes
}

func identityStr(id *pgd.Identity) string {
	if id == nil {
		return ""
	}
	s := id.Generated
	if id.Sequence != nil {
		s += fmt.Sprintf(":%d:%d:%d:%d:%d:%s",
			id.Sequence.Start, id.Sequence.Increment, id.Sequence.Min,
			id.Sequence.Max, id.Sequence.Cache, id.Sequence.Cycle)
	}
	return s
}

func seqOptStr(seq *pgd.IdentitySeqOpt) string {
	if seq == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d:%d:%d:%d:%s", seq.Start, seq.Increment, seq.Min, seq.Max, seq.Cache, seq.Cycle)
}

func indexElemKey(name, order, nulls, opclass string) string {
	s := name
	if order != "" && order != "asc" {
		s += ":" + order
	}
	if nulls != "" {
		s += ":" + nulls
	}
	if opclass != "" {
		s += ":" + opclass
	}
	return s
}

func fkColNames(cols []pgd.FKCol) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

// indexIdentity identifies what an index indexes. Indexes with equal identities are
// the same index, possibly renamed.
func indexIdentity(idx *pgd.Index) string {
	return indexIdentityExpr(idx, strings.TrimSpace)
}

// indexIdentityExpr is indexIdentity with expressions and the predicate converted by expr.
func indexIdentityExpr(idx *pgd.Index, expr func(string) string) string {
	keys := make([]string, len(idx.Keys))
	for i, k := range idx.Keys {
		id := k.ID()
		if k.IsExpression() {
			id = expr(id)
		}
		keys[i] = indexElemKey(id, k.Order, k.Nulls, k.Opclass)
	}
	where := ""
	if idx.Where != nil {
		where = expr(idx.Where.Value)
	}
	using := strings.ToLower(cmp.Or(idx.Using, "btree"))
	return strings.Join([]string{idx.Table, strconv.FormatBool(idx.Unique == "true"), using,
		strings.Join(keys, ","), where}, "|")
}

// indexChanged reports whether the index DDL changed, apart from its name. Expressions
// that differ only in formatting are equal.
func indexChanged(old, updated *pgd.Index) bool {
	if indexOptions(old) != indexOptions(updated) {
		return true
	}
	return indexIdentity(old) != indexIdentity(updated) &&
		indexIdentityExpr(old, sqlconv.ExprKey) != indexIdentityExpr(updated, sqlconv.ExprKey)
}

// indexOptions returns INCLUDE, NULLS NOT DISTINCT, WITH and TABLESPACE of an index.
func indexOptions(idx *pgd.Index) string {
	var include string
	if idx.Include != nil {
		include = strings.Join(pgd.ColRefNames(idx.Include.Columns), ",")
	}
	var with []string
	if idx.With != nil {
		for _, p := range idx.With.Params {
			with = append(with, p.Name+"="+p.Value)
		}
	}
	return strings.Join([]string{include, strconv.FormatBool(idx.NullsDistinct == "false"),
		strings.Join(with, ","), idx.Tablespace}, "|")
}

func fkChanged(old, updated *pgd.ForeignKey) bool {
	if old.ToTable != updated.ToTable {
		return true
	}
	if old.OnDelete != updated.OnDelete || old.OnUpdate != updated.OnUpdate {
		return true
	}
	if len(old.Columns) != len(updated.Columns) {
		return true
	}
	for i := range old.Columns {
		if old.Columns[i].Name != updated.Columns[i].Name || old.Columns[i].References != updated.Columns[i].References {
			return true
		}
	}
	return false
}

func pkChanged(old, updated *pgd.PrimaryKey) bool {
	return !slices.Equal(pgd.ColRefNames(old.Columns), pgd.ColRefNames(updated.Columns))
}

func isCompatibleCast(old, updated string) bool {
	// varchar(N) → varchar(M) where M > N is safe
	if strings.HasPrefix(old, "varchar") && strings.HasPrefix(updated, "varchar") {
		return true
	}
	// integer → bigint is safe
	if old == "integer" && updated == "bigint" {
		return true
	}
	if old == "smallint" && (updated == "integer" || updated == "bigint") {
		return true
	}
	return false
}

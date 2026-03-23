package diff

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// tstzRe matches timestamp with timezone offset in partition bounds.
var tstzRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}[+-]\d{2}`)

// DiffResult holds the list of changes between two Projects.
type DiffResult struct {
	Changes []Change
}

// SQL returns the full migration script.
func (r *DiffResult) SQL() string {
	var b strings.Builder
	for _, c := range r.Changes {
		b.WriteString(c.SQL)
		b.WriteString("\n\n")
	}
	return b.String()
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
	b := &diffBuilder{}

	oldSchemas := schemaMap(old.Schemas)
	newSchemas := schemaMap(updated.Schemas)

	// Phase 1: DROP (reverse dependency order)
	// drop FK first, then indexes, then columns, then tables
	for name, os := range oldSchemas {
		ns := newSchemas[name]
		b.schema = name
		b.dropPhase(os, ns)
	}

	// Phase 2: CREATE/ALTER (forward dependency order)
	for name, ns := range newSchemas {
		os := oldSchemas[name]
		b.schema = name
		b.createPhase(os, ns)
	}

	// Phase 3: Enums
	if old.Types != nil || updated.Types != nil {
		var oldEnums, newEnums []pgd.Enum
		if old.Types != nil {
			oldEnums = old.Types.Enums
		}
		if updated.Types != nil {
			newEnums = updated.Types.Enums
		}
		b.diffEnums(oldEnums, newEnums)
	}

	return &DiffResult{Changes: b.changes}
}

// diffBuilder accumulates changes.
type diffBuilder struct {
	schema  string
	changes []Change
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

func (b *diffBuilder) dropPhase(old, updated *pgd.Schema) {
	if old == nil {
		return
	}
	newTables := tableMap(nilTables(updated))
	oldTables := tableMap(old.Tables)

	// drop FK on tables that still exist but lost FK
	for _, ot := range old.Tables {
		nt, exists := newTables[ot.Name]
		if !exists {
			continue
		}
		b.dropRemovedFK(ot.Name, ot.FKs, nt.FKs)
		b.dropRemovedChecks(ot.Name, ot.Checks, nt.Checks)
		b.dropRemovedUniques(ot.Name, ot.Uniques, nt.Uniques)
		b.dropModifiedPK(ot.Name, ot.PK, nt.PK)
	}

	// drop indexes
	newIndexes := indexMap(nilIndexes(updated))
	updatedIndexes := nilIndexes(updated)
	for _, oi := range old.Indexes {
		if ni, exists := newIndexes[oi.Name]; exists {
			if indexChanged(&oi, ni) {
				b.dropIndex(oi.Name)
			}
			continue
		}
		// fallback: match by semantic key (table + columns + using + where)
		if matchIndexBySemantic(&oi, updatedIndexes) != nil {
			continue
		}
		b.dropIndex(oi.Name)
	}

	// drop columns on existing tables
	for _, ot := range old.Tables {
		nt, exists := newTables[ot.Name]
		if !exists {
			continue
		}
		b.dropRemovedColumns(ot.Name, ot.Columns, nt.Columns)
	}

	// drop tables
	for name := range oldTables {
		if _, exists := newTables[name]; !exists {
			b.dropTable(name)
		}
	}
}

// --- Phase 2: CREATE/ALTER ---

func (b *diffBuilder) createPhase(old, updated *pgd.Schema) {
	if updated == nil {
		return
	}
	oldTables := tableMap(nilTables(old))

	// add new tables
	for _, nt := range updated.Tables {
		if _, exists := oldTables[nt.Name]; !exists {
			b.addTable(&nt)
		}
	}

	// alter existing tables (columns + table comment)
	for _, nt := range updated.Tables {
		ot, exists := oldTables[nt.Name]
		if !exists {
			continue
		}
		b.diffColumns(nt.Name, ot.Columns, nt.Columns)
		b.diffTableComment(nt.Name, ot.Comment, nt.Comment)
	}

	// add/recreate indexes
	oldIndexes := indexMap(nilIndexes(old))
	oldIdxSlice := nilIndexes(old)
	for _, ni := range updated.Indexes {
		if oi, exists := oldIndexes[ni.Name]; exists {
			if indexChanged(oi, &ni) {
				b.addIndex(&ni)
			}
			continue
		}
		// fallback: match by semantic key
		if match := matchIndexBySemantic(&ni, oldIdxSlice); match != nil {
			if indexChanged(match, &ni) {
				b.addIndex(&ni)
			}
			continue
		}
		b.addIndex(&ni)
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
		b.addNewFK(nt.Name, ot.FKs, nt.FKs)
		b.diffPartitions(nt.Name, ot.Partitions, nt.Partitions)
		b.diffPartitionBy(nt.Name, ot.PartitionBy, nt.PartitionBy)
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
}

func (b *diffBuilder) dropTable(name string) {
	b.add(Change{
		Object: "table", Action: "drop", Name: name,
		SQL:     fmt.Sprintf("DROP TABLE %s;", b.qt(name)),
		Hazards: []Hazard{{Level: "dangerous", Code: "DELETES_DATA", Message: fmt.Sprintf("drops table %s and all its data", name)}},
	})
}

// --- Columns ---

func (b *diffBuilder) diffTableComment(table, oldComment, newComment string) {
	if oldComment == newComment {
		return
	}
	t := b.qt(table)
	if newComment == "" {
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL: fmt.Sprintf("COMMENT ON TABLE %s IS NULL;", t),
		})
	} else {
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL: fmt.Sprintf("COMMENT ON TABLE %s IS %s;", t, pgd.EscapeComment(newComment)),
		})
	}
}

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
	if c.Nullable == "false" && c.Default == "" && c.Identity == nil {
		hazards = append(hazards, Hazard{
			Level: "warning", Code: "BACKFILL_REQUIRED",
			Message: fmt.Sprintf("column %s is NOT NULL without DEFAULT", c.Name),
		})
	}

	b.add(Change{
		Object: "column", Action: "add", Table: table, Name: c.Name,
		SQL: sql, Hazards: hazards,
	})
}

func (b *diffBuilder) dropColumn(table, col string) {
	b.add(Change{
		Object: "column", Action: "drop", Table: table, Name: col,
		SQL:     fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", b.qt(table), b.q(col)),
		Hazards: []Hazard{{Level: "dangerous", Code: "DELETES_DATA", Message: fmt.Sprintf("drops column %s.%s and all its data", table, col)}},
	})
}

func (b *diffBuilder) diffColumn(table string, old, updated *pgd.Column) {
	t := b.qt(table)
	col := b.q(updated.Name)

	b.diffColumnType(table, t, col, old, updated)
	b.diffColumnNullable(table, t, col, old, updated)
	b.diffColumnDefault(table, t, col, old, updated)
	b.diffColumnIdentity(table, t, col, old, updated)
	b.diffColumnComment(table, t, col, old, updated)
	b.diffColumnCompression(table, t, col, old, updated)
	b.diffColumnStorage(table, t, col, old, updated)
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
	if old.Default == updated.Default {
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

func (b *diffBuilder) diffColumnComment(table, t, col string, old, updated *pgd.Column) {
	if old.Comment == updated.Comment {
		return
	}
	if updated.Comment == "" {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("COMMENT ON COLUMN %s.%s IS NULL;", t, col),
		})
	} else {
		b.add(Change{
			Object: "column", Action: "alter", Table: table, Name: updated.Name,
			SQL: fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s;", t, col, pgd.EscapeComment(updated.Comment)),
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
			SQL:     fmt.Sprintf("-- Cannot add PARTITION BY to existing table %s; requires table recreation", t),
			Hazards: []Hazard{{Level: "dangerous", Code: "TABLE_RECREATE", Message: "adding PARTITION BY requires table recreation"}},
		})
	case old != nil && updated == nil:
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL:     fmt.Sprintf("-- Cannot remove PARTITION BY from table %s; requires table recreation", t),
			Hazards: []Hazard{{Level: "dangerous", Code: "TABLE_RECREATE", Message: "removing PARTITION BY requires table recreation"}},
		})
	default:
		b.add(Change{
			Object: "table", Action: "alter", Table: table, Name: table,
			SQL:     fmt.Sprintf("-- Cannot change PARTITION BY strategy on table %s; requires table recreation", t),
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
				SQL:     fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s;", t, b.q(op.Name)),
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
				SQL: fmt.Sprintf("CREATE TABLE %s PARTITION OF %s\n    %s;", b.q(np.Name), t, np.Bound),
			})
			continue
		}
		// Bound changed → detach + attach (normalize timestamps for TZ-aware comparison)
		if normalizeBound(op.Bound) != normalizeBound(np.Bound) {
			b.add(Change{
				Object: "partition", Action: "alter", Table: table, Name: np.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s;\nALTER TABLE %s ATTACH PARTITION %s %s;",
					t, b.q(np.Name), t, b.q(np.Name), np.Bound),
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
		SQL: fmt.Sprintf("DROP INDEX %s;", b.q(name)),
	})
}

// --- FK ---

func (b *diffBuilder) dropRemovedFK(table string, old, updated []pgd.ForeignKey) {
	newByName := fkMap(updated)
	for _, ofk := range old {
		if nfk, exists := newByName[ofk.Name]; exists {
			if fkChanged(&ofk, nfk) {
				b.dropFK(table, ofk.Name)
			}
			continue
		}
		// fallback: match by semantic key (columns + target table)
		if matchFKBySemantic(&ofk, updated) != nil {
			continue
		}
		b.dropFK(table, ofk.Name)
	}
}

func (b *diffBuilder) addNewFK(table string, old, updated []pgd.ForeignKey) {
	oldByName := fkMap(old)
	for _, nfk := range updated {
		if ofk, exists := oldByName[nfk.Name]; exists {
			if fkChanged(ofk, &nfk) {
				b.addFK(table, &nfk)
			}
			continue
		}
		// fallback: match by semantic key (columns + target table)
		if match := matchFKBySemantic(&nfk, old); match != nil {
			if fkChanged(match, &nfk) {
				b.addFK(table, &nfk)
			}
			continue
		}
		b.addFK(table, &nfk)
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

func (b *diffBuilder) dropFK(table, name string) {
	b.add(Change{
		Object: "fk", Action: "drop", Table: table, Name: name,
		SQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.qt(table), b.q(name)),
	})
}

// --- PK ---

func (b *diffBuilder) dropModifiedPK(table string, old, updated *pgd.PrimaryKey) {
	if old == nil {
		return
	}
	if updated == nil || pkChanged(old, updated) {
		b.add(Change{
			Object: "pk", Action: "drop", Table: table, Name: old.Name,
			SQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.qt(table), b.q(old.Name)),
		})
	}
}

func (b *diffBuilder) addNewPK(table string, old, updated *pgd.PrimaryKey) {
	if updated == nil {
		return
	}
	if old == nil || pkChanged(old, updated) {
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
		if _, exists := newU[ou.Name]; !exists {
			b.add(Change{
				Object: "unique", Action: "drop", Table: table, Name: ou.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.qt(table), b.q(ou.Name)),
			})
		}
	}
}

func (b *diffBuilder) addNewUniques(table string, old, updated []pgd.Unique) {
	oldU := uniqueMap(old)
	for _, nu := range updated {
		if _, exists := oldU[nu.Name]; !exists {
			cols := pgd.QuotedColList(nu.Columns)
			b.add(Change{
				Object: "unique", Action: "add", Table: table, Name: nu.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);", b.qt(table), b.q(nu.Name), cols),
			})
		}
	}
}

// --- CHECK ---

func (b *diffBuilder) dropRemovedChecks(table string, old, updated []pgd.Check) {
	newC := checkMap(updated)
	for _, oc := range old {
		if _, exists := newC[oc.Name]; !exists {
			b.add(Change{
				Object: "check", Action: "drop", Table: table, Name: oc.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", b.qt(table), b.q(oc.Name)),
			})
		}
	}
}

func (b *diffBuilder) addNewChecks(table string, old, updated []pgd.Check) {
	oldC := checkMap(old)
	for _, nc := range updated {
		if _, exists := oldC[nc.Name]; !exists {
			b.add(Change{
				Object: "check", Action: "add", Table: table, Name: nc.Name,
				SQL: fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s);", b.qt(table), b.q(nc.Name), nc.Expression),
			})
		}
	}
}

// --- Enums ---

func (b *diffBuilder) diffEnums(old, updated []pgd.Enum) {
	oldMap := make(map[string]*pgd.Enum, len(old))
	for i := range old {
		oldMap[old[i].Name] = &old[i]
	}

	for _, ne := range updated {
		oe, exists := oldMap[ne.Name]
		if !exists {
			continue // new enum = handled by CREATE TYPE in DDL
		}
		oldLabels := make(map[string]bool, len(oe.Labels))
		for _, l := range oe.Labels {
			oldLabels[l] = true
		}
		for _, l := range ne.Labels {
			if !oldLabels[l] {
				b.add(Change{
					Object: "enum", Action: "alter", Name: ne.Name,
					SQL: fmt.Sprintf("ALTER TYPE %s ADD VALUE '%s';", b.q(ne.Name), l),
				})
			}
		}
	}
}

// --- helpers ---

func schemaMap(schemas []pgd.Schema) map[string]*pgd.Schema {
	m := make(map[string]*pgd.Schema, len(schemas))
	for i := range schemas {
		m[schemas[i].Name] = &schemas[i]
	}
	return m
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

// matchIndexBySemantic finds an index in candidates with the same table, columns, using, and where.
func matchIndexBySemantic(idx *pgd.Index, candidates []pgd.Index) *pgd.Index {
	for i := range candidates {
		c := &candidates[i]
		if c.Table != idx.Table || c.Unique != idx.Unique || c.Using != idx.Using {
			continue
		}
		if colRefNames(c.Columns) != colRefNames(idx.Columns) {
			continue
		}
		cWhere, iWhere := "", ""
		if c.Where != nil {
			cWhere = c.Where.Value
		}
		if idx.Where != nil {
			iWhere = idx.Where.Value
		}
		if cWhere != iWhere {
			continue
		}
		return c
	}
	return nil
}

func indexMap(indexes []pgd.Index) map[string]*pgd.Index {
	m := make(map[string]*pgd.Index, len(indexes))
	for i := range indexes {
		m[indexes[i].Name] = &indexes[i]
	}
	return m
}

// matchFKBySemantic finds a FK in candidates that has the same columns and target table.
func matchFKBySemantic(fk *pgd.ForeignKey, candidates []pgd.ForeignKey) *pgd.ForeignKey {
	for i := range candidates {
		c := &candidates[i]
		if c.ToTable != fk.ToTable || len(c.Columns) != len(fk.Columns) {
			continue
		}
		match := true
		for j := range fk.Columns {
			if fk.Columns[j].Name != c.Columns[j].Name || fk.Columns[j].References != c.Columns[j].References {
				match = false
				break
			}
		}
		if match {
			return c
		}
	}
	return nil
}

func fkMap(fks []pgd.ForeignKey) map[string]*pgd.ForeignKey {
	m := make(map[string]*pgd.ForeignKey, len(fks))
	for i := range fks {
		m[fks[i].Name] = &fks[i]
	}
	return m
}

func uniqueMap(us []pgd.Unique) map[string]*pgd.Unique {
	m := make(map[string]*pgd.Unique, len(us))
	for i := range us {
		m[us[i].Name] = &us[i]
	}
	return m
}

func checkMap(cs []pgd.Check) map[string]*pgd.Check {
	m := make(map[string]*pgd.Check, len(cs))
	for i := range cs {
		m[cs[i].Name] = &cs[i]
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

func colRefNames(refs []pgd.ColRef) string {
	var parts []string
	for _, r := range refs {
		s := r.Name
		if r.Order != "" {
			s += ":" + r.Order
		}
		if r.Nulls != "" {
			s += ":" + r.Nulls
		}
		if r.Opclass != "" {
			s += ":" + r.Opclass
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}

func indexChanged(old, updated *pgd.Index) bool {
	if old.Unique != updated.Unique {
		return true
	}
	if old.Using != updated.Using {
		return true
	}
	if colRefNames(old.Columns) != colRefNames(updated.Columns) {
		return true
	}
	oldWhere := ""
	if old.Where != nil {
		oldWhere = old.Where.Value
	}
	newWhere := ""
	if updated.Where != nil {
		newWhere = updated.Where.Value
	}
	if oldWhere != newWhere {
		return true
	}
	return false
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
	return colRefNames(old.Columns) != colRefNames(updated.Columns)
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

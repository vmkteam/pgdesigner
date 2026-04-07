package rpc

import (
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/designer/lint"
	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func newERDSchema(src pgd.ERDSchema) ERDSchema {
	return ERDSchema{
		Tables:     NewERDTables(src.Tables),
		References: NewERDReferences(src.References),
	}
}

func newObjectItems(p *pgd.Project) []ObjectItem {
	defaultSchema := p.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	// erdKey returns qualified name for non-default schemas, plain name otherwise.
	erdKey := func(schemaName, tableName string) string {
		if schemaName == defaultSchema {
			return tableName
		}
		return schemaName + "." + tableName
	}

	var items []ObjectItem

	for _, schema := range p.Schemas {
		for _, t := range schema.Tables {
			key := erdKey(schema.Name, t.Name)
			items = append(items, ObjectItem{Name: t.Name, Kind: "table", Table: key})
			for _, c := range t.Columns {
				items = append(items, ObjectItem{Name: t.Name + "." + c.Name, Kind: "column", Table: key})
			}
			if t.PK != nil && t.PK.Name != "" {
				items = append(items, ObjectItem{Name: t.PK.Name, Kind: "pk", Table: key})
			}
			for _, u := range t.Uniques {
				items = append(items, ObjectItem{Name: u.Name, Kind: "unique", Table: key})
			}
			for _, ch := range t.Checks {
				items = append(items, ObjectItem{Name: ch.Name, Kind: "check", Table: key})
			}
			for _, fk := range t.FKs {
				items = append(items, ObjectItem{Name: fk.Name, Kind: "fk", Table: key})
			}
		}
		for _, idx := range schema.Indexes {
			items = append(items, ObjectItem{Name: idx.Name, Kind: "index", Table: erdKey(schema.Name, idx.Table)})
		}
	}

	for _, tr := range p.Triggers {
		items = append(items, ObjectItem{Name: tr.Name, Kind: "trigger", Table: tr.Table})
	}
	for _, seq := range p.Sequences {
		items = append(items, ObjectItem{Name: seq.Name, Kind: "sequence", Table: ""})
	}
	for _, ext := range p.Extensions {
		items = append(items, ObjectItem{Name: ext.Name, Kind: "extension", Table: ""})
	}
	for _, fn := range p.Functions {
		items = append(items, ObjectItem{Name: fn.Name, Kind: "function", Table: ""})
	}
	if p.Views != nil {
		for _, v := range p.Views.Views {
			items = append(items, ObjectItem{Name: v.Name, Kind: "view", Table: ""})
		}
		for _, mv := range p.Views.MatViews {
			items = append(items, ObjectItem{Name: mv.Name, Kind: "matview", Table: ""})
		}
	}
	if p.Types != nil {
		for _, e := range p.Types.Enums {
			items = append(items, ObjectItem{Name: e.Name, Kind: "enum", Table: ""})
		}
		for _, d := range p.Types.Domains {
			items = append(items, ObjectItem{Name: d.Name, Kind: "domain", Table: ""})
		}
		for _, c := range p.Types.Composites {
			items = append(items, ObjectItem{Name: c.Name, Kind: "composite", Table: ""})
		}
	}

	return items
}

func newTableDetail(p *pgd.Project, t *pgd.Table, schema *pgd.Schema) *TableDetail { //nolint:gocognit,gocyclo,cyclop
	// Build PK and FK column sets
	pkCols := map[string]bool{}
	if t.PK != nil {
		for _, c := range t.PK.Columns {
			pkCols[c.Name] = true
		}
	}
	fkCols := map[string]bool{}
	for _, fk := range t.FKs {
		for _, c := range fk.Columns {
			fkCols[c.Name] = true
		}
	}

	td := &TableDetail{
		Name:       t.Name,
		Schema:     schema.Name,
		Unlogged:   t.Unlogged == "true",
		Tablespace: t.Tablespace,
		Comment:    t.Comment,
	}

	// Columns
	for _, c := range t.Columns {
		col := ColumnDetail{
			Name:        c.Name,
			Type:        c.Type,
			Length:      c.Length,
			Precision:   c.Precision,
			Scale:       c.Scale,
			Nullable:    c.Nullable != "false",
			Default:     c.Default,
			PK:          pkCols[c.Name],
			FK:          fkCols[c.Name],
			Comment:     c.Comment,
			Compression: c.Compression,
			Storage:     c.Storage,
			Collation:   c.Collation,
		}
		if c.Identity != nil {
			col.Identity = c.Identity.Generated
			if c.Identity.Sequence != nil {
				col.IdentitySeqOpt = &IdentitySeqOpt{
					Start: c.Identity.Sequence.Start, Increment: c.Identity.Sequence.Increment,
					Min: c.Identity.Sequence.Min, Max: c.Identity.Sequence.Max,
					Cache: c.Identity.Sequence.Cache, Cycle: c.Identity.Sequence.Cycle == "true",
				}
			}
		}
		if c.Generated != nil {
			col.Generated = c.Generated.Expression
			col.GeneratedStored = c.Generated.Stored != "false"
		}
		td.Columns = append(td.Columns, col)
	}

	// PK
	if t.PK != nil {
		pk := &PKDetail{Name: t.PK.Name}
		for _, c := range t.PK.Columns {
			pk.Columns = append(pk.Columns, c.Name)
		}
		td.PK = pk
	}

	// Uniques
	for _, u := range t.Uniques {
		ud := UniqueDetail{Name: u.Name, NullsDistinct: u.NullsDistinct == "true"}
		for _, c := range u.Columns {
			ud.Columns = append(ud.Columns, c.Name)
		}
		td.Uniques = append(td.Uniques, ud)
	}

	// Checks
	for _, ch := range t.Checks {
		td.Checks = append(td.Checks, CheckDetail{Name: ch.Name, Expression: ch.Expression})
	}

	// Excludes
	for _, ex := range t.Excludes {
		ed := ExcludeDetail{Name: ex.Name, Using: ex.Using}
		if ex.Where != nil {
			ed.Where = ex.Where.Value
		}
		for _, el := range ex.Elements {
			ed.Elements = append(ed.Elements, ExcludeElementDetail{Column: el.Column, Expression: el.Expression, Opclass: el.Opclass, With: el.With})
		}
		td.Excludes = append(td.Excludes, ed)
	}

	// FKs
	for _, fk := range t.FKs {
		fd := FKDetail{
			Name:       fk.Name,
			ToTable:    fk.ToTable,
			OnDelete:   fk.OnDelete,
			OnUpdate:   fk.OnUpdate,
			Deferrable: fk.Deferrable == "true",
			Initially:  fk.Initially,
		}
		for _, c := range fk.Columns {
			fd.Columns = append(fd.Columns, FKColDetail{Name: c.Name, References: c.References})
		}
		td.FKs = append(td.FKs, fd)
	}

	// Indexes (from schema-level)
	for _, idx := range schema.Indexes {
		if idx.Table != t.Name {
			continue
		}
		id := IndexDetail{
			Name:          idx.Name,
			Unique:        idx.Unique == "true",
			NullsDistinct: idx.NullsDistinct == "true",
			Using:         idx.Using,
		}
		for _, c := range idx.Columns {
			id.Columns = append(id.Columns, IndexColDetail{Name: c.Name, Order: c.Order, Nulls: c.Nulls, Opclass: c.Opclass})
		}
		for _, e := range idx.Expressions {
			id.Expressions = append(id.Expressions, e.Value)
		}
		if idx.With != nil {
			for _, p := range idx.With.Params {
				id.With = append(id.With, WithParamDetail{Name: p.Name, Value: p.Value})
			}
		}
		if idx.Where != nil {
			id.Where = idx.Where.Value
		}
		if idx.Include != nil {
			for _, c := range idx.Include.Columns {
				id.Include = append(id.Include, c.Name)
			}
		}
		td.Indexes = append(td.Indexes, id)
	}

	// Partition info
	if t.PartitionBy != nil {
		pb := &PartitionByRPC{Type: t.PartitionBy.Type}
		for _, c := range t.PartitionBy.Columns {
			pb.Columns = append(pb.Columns, c.Name)
		}
		td.PartitionBy = pb
	}
	for _, p := range t.Partitions {
		td.Partitions = append(td.Partitions, PartitionRPC{Name: p.Name, Bound: p.Bound})
	}

	// DDL preview
	td.DDL = pgd.GenerateTableDDL(p, t.Name)

	return td
}

func applyGeneralToTable(t *pgd.Table, general *GeneralInput) {
	if general == nil {
		return
	}
	if general.Name != nil {
		t.Name = *general.Name
	}
	if general.Comment != nil {
		t.Comment = *general.Comment
	}
	if general.Unlogged != nil {
		if *general.Unlogged {
			t.Unlogged = "true"
		} else {
			t.Unlogged = ""
		}
	}
}

// newSchemaFragment creates a minimal schema containing only the given table and its indexes.
func newSchemaFragment(schema *pgd.Schema, table *pgd.Table) pgd.Schema {
	s := pgd.Schema{
		Name:   schema.Name,
		Tables: []pgd.Table{*table},
	}
	for _, idx := range schema.Indexes {
		if idx.Table == table.Name {
			s.Indexes = append(s.Indexes, idx)
		}
	}
	return s
}

// newTableCopy returns a shallow copy of a table (slices are shared but that's OK for read-only use).
func newTableCopy(t *pgd.Table) pgd.Table {
	return *t
}

func applyPartitions(st *store.ProjectStore, name string, partitionBy *PartitionByRPC, partitions []PartitionRPC) error {
	if partitionBy == nil && partitions == nil {
		return nil
	}
	var pb *pgd.PartitionBy
	if partitionBy != nil {
		pb = &pgd.PartitionBy{Type: partitionBy.Type}
		for _, c := range partitionBy.Columns {
			pb.Columns = append(pb.Columns, pgd.ColRef{Name: c})
		}
	}
	var parts []pgd.Partition
	for _, p := range partitions {
		parts = append(parts, pgd.Partition{Name: p.Name, Bound: p.Bound})
	}
	return st.UpdateTablePartitions(name, pb, parts)
}

func ruleTitle(code string) string {
	if r, ok := lint.Rules[code]; ok {
		return r.Title
	}
	return code
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, r := range strings.Split(s, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			result = append(result, r)
		}
	}
	return result
}

package sql

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	pgquery "github.com/wasilibs/go-pgquery"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Convert parses PostgreSQL DDL statements and returns a pgd.Project.
func Convert(data []byte, name string) (*pgd.Project, error) {
	// Strip UTF-8 BOM if present.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	return ParseSQL(string(data), name)
}

// ParseSQL parses PostgreSQL DDL statements and returns a pgd.Project.
func ParseSQL(sql string, projectName string) (*pgd.Project, error) {
	sql = cleanDump(sql)

	result, err := pgquery.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse sql: %w", err)
	}

	c := &sqlConverter{
		schemas: make(map[string]*pgd.Schema),
	}

	for _, rawStmt := range result.Stmts {
		c.convertStmt(rawStmt.Stmt)
	}

	p := &pgd.Project{
		Version:       1,
		PgVersion:     "18",
		DefaultSchema: "public",
		ProjectMeta: pgd.ProjectMeta{
			Name: projectName,
			Settings: pgd.Settings{
				Naming:   pgd.Naming{Convention: "camelCase"},
				Defaults: pgd.Defaults{Nullable: "true", OnDelete: "restrict", OnUpdate: "restrict"},
			},
		},
		Extensions: c.extensions,
		Sequences:  c.sequences,
		Schemas:    c.schemaOrder(),
		Functions:  c.functions,
		Triggers:   c.triggers,
		Comments:   c.comments,
		Layouts:    pgd.Layouts{Layouts: []pgd.Layout{{Name: "Default Diagram", Default: "true"}}},
	}

	if len(c.enums) > 0 || len(c.composites) > 0 || len(c.domains) > 0 {
		p.Types = &pgd.Types{
			Enums:      c.enums,
			Composites: c.composites,
			Domains:    c.domains,
		}
	}

	if len(c.views) > 0 || len(c.matViews) > 0 {
		p.Views = &pgd.Views{
			Views:    c.views,
			MatViews: c.matViews,
		}
	}

	// Post-process: resolve FK columns that reference PK implicitly (no column list in REFERENCES).
	resolveFKImplicitPK(p)

	// Deduplicate indexes with the same table and columns (pg_dump may export both
	// explicitly created and auto-inherited partition indexes).
	deduplicateIndexes(p)

	// Remove indexes on partition children — they are auto-inherited from parent.
	removePartitionChildIndexes(p)

	// Convert Variant A partitions (separate tables with PartitionOf) to Variant B (nested).
	pgd.MigratePartitions(p)

	// Normalize ordering for stable round-trip (tables, FK, indexes by name).
	normalizeOrder(p)

	return p, nil
}

func normalizeOrder(p *pgd.Project) {
	for si := range p.Schemas {
		sort.Slice(p.Schemas[si].Tables, func(i, j int) bool {
			return p.Schemas[si].Tables[i].Name < p.Schemas[si].Tables[j].Name
		})
		sort.Slice(p.Schemas[si].Indexes, func(i, j int) bool {
			return p.Schemas[si].Indexes[i].Name < p.Schemas[si].Indexes[j].Name
		})
		for ti := range p.Schemas[si].Tables {
			sort.Slice(p.Schemas[si].Tables[ti].FKs, func(i, j int) bool {
				return p.Schemas[si].Tables[ti].FKs[i].Name < p.Schemas[si].Tables[ti].FKs[j].Name
			})
		}
	}
}

// resolveFKImplicitPK fixes FK columns where REFERENCES table had no column list.
// In that case the parser stored fkCol.References = fkCol.Name (fallback).
// We resolve to the actual PK columns of the target table.
func resolveFKImplicitPK(p *pgd.Project) {
	pkMap := buildPKMap(p)
	for si := range p.Schemas {
		for ti := range p.Schemas[si].Tables {
			for fi := range p.Schemas[si].Tables[ti].FKs {
				fixFKImplicitRefs(&p.Schemas[si].Tables[ti].FKs[fi], pkMap)
			}
		}
	}
}

// deduplicateIndexes removes duplicate indexes on the same table+columns.
// pg_dump may export both explicitly created and auto-inherited partition indexes.
func deduplicateIndexes(p *pgd.Project) {
	for si := range p.Schemas {
		seen := make(map[string]bool)
		var filtered []pgd.Index
		for _, idx := range p.Schemas[si].Indexes {
			key := indexKey(&idx)
			if seen[key] {
				continue
			}
			seen[key] = true
			filtered = append(filtered, idx)
		}
		p.Schemas[si].Indexes = filtered
	}
}

// removePartitionChildIndexes drops indexes on tables that have PartitionOf set.
// These indexes are auto-inherited from the parent table.
func removePartitionChildIndexes(p *pgd.Project) {
	for si := range p.Schemas {
		children := make(map[string]bool)
		for _, t := range p.Schemas[si].Tables {
			if t.PartitionOf != "" {
				children[t.Name] = true
			}
		}
		if len(children) == 0 {
			continue
		}
		var filtered []pgd.Index
		for _, idx := range p.Schemas[si].Indexes {
			if !children[idx.Table] {
				filtered = append(filtered, idx)
			}
		}
		p.Schemas[si].Indexes = filtered
	}
}

func indexKey(idx *pgd.Index) string {
	var b strings.Builder
	b.WriteString(idx.Table)
	b.WriteByte(':')
	b.WriteString(idx.Using)
	b.WriteByte(':')
	b.WriteString(idx.Unique)
	b.WriteByte(':')
	for i, c := range idx.Columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(c.Name)
	}
	for i, e := range idx.Expressions {
		if i > 0 || len(idx.Columns) > 0 {
			b.WriteByte(',')
		}
		b.WriteString(e.Value)
	}
	if idx.Where != nil {
		b.WriteByte(':')
		b.WriteString(idx.Where.Value)
	}
	return b.String()
}

// buildPKMap returns tableName → []pkColName (including schema-qualified).
func buildPKMap(p *pgd.Project) map[string][]string {
	pkMap := make(map[string][]string)
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			if t.PK == nil {
				continue
			}
			cols := make([]string, len(t.PK.Columns))
			for i, c := range t.PK.Columns {
				cols[i] = c.Name
			}
			pkMap[t.Name] = cols
			if s.Name != "" && s.Name != "public" {
				pkMap[s.Name+"."+t.Name] = cols
			}
		}
	}
	return pkMap
}

// fixFKImplicitRefs fixes a single FK whose References were set to FK col names (fallback).
func fixFKImplicitRefs(fk *pgd.ForeignKey, pkMap map[string][]string) {
	if len(fk.Columns) == 0 {
		return
	}
	pkCols, ok := pkMap[fk.ToTable]
	if !ok || len(pkCols) != len(fk.Columns) {
		return
	}
	needsFix := false
	for i, c := range fk.Columns {
		if c.References == c.Name && c.Name != pkCols[i] {
			needsFix = true
			break
		}
	}
	if needsFix {
		for i := range fk.Columns {
			if i < len(pkCols) {
				fk.Columns[i].References = pkCols[i]
			}
		}
	}
}

type sqlConverter struct {
	schemas    map[string]*pgd.Schema
	views      []pgd.View
	matViews   []pgd.MaterializedView
	functions  []pgd.Function
	sequences  []pgd.Sequence
	extensions []pgd.Extension
	enums      []pgd.Enum
	composites []pgd.Composite
	domains    []pgd.Domain
	triggers   []pgd.Trigger
	comments   []pgd.Comment
}

func (c *sqlConverter) getSchema(name string) *pgd.Schema {
	if name == "" {
		name = "public"
	}
	if s, ok := c.schemas[name]; ok {
		return s
	}
	c.schemas[name] = &pgd.Schema{Name: name}
	return c.schemas[name]
}

// schemaOrder returns schemas sorted: "public" first, then alphabetical.
func (c *sqlConverter) schemaOrder() []pgd.Schema {
	var names []string
	for name := range c.schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	// move "public" to front
	for i, n := range names {
		if n == "public" && i > 0 {
			names = append([]string{"public"}, append(names[:i], names[i+1:]...)...)
			break
		}
	}
	var result []pgd.Schema
	for _, name := range names {
		result = append(result, *c.schemas[name])
	}
	return result
}

func (c *sqlConverter) convertStmt(node *pg.Node) {
	switch n := node.Node.(type) {
	case *pg.Node_CreateStmt:
		c.convertCreateTable(n.CreateStmt)
	case *pg.Node_IndexStmt:
		c.convertCreateIndex(n.IndexStmt)
	case *pg.Node_AlterTableStmt:
		c.convertAlterTable(n.AlterTableStmt)
	case *pg.Node_ViewStmt:
		c.convertView(n.ViewStmt)
	case *pg.Node_CreateTableAsStmt:
		c.convertMatView(n.CreateTableAsStmt)
	case *pg.Node_CreateFunctionStmt:
		c.convertFunction(n.CreateFunctionStmt)
	case *pg.Node_CreateSeqStmt:
		c.convertSequence(n.CreateSeqStmt)
	case *pg.Node_CreateExtensionStmt:
		c.convertExtension(n.CreateExtensionStmt)
	case *pg.Node_CreateEnumStmt:
		c.convertEnum(n.CreateEnumStmt)
	case *pg.Node_CompositeTypeStmt:
		c.convertComposite(n.CompositeTypeStmt)
	case *pg.Node_CreateDomainStmt:
		c.convertDomain(n.CreateDomainStmt)
	case *pg.Node_CreateTrigStmt:
		c.convertTrigger(n.CreateTrigStmt)
	case *pg.Node_CommentStmt:
		c.convertComment(n.CommentStmt)
	case *pg.Node_CreateSchemaStmt:
		c.getSchema(n.CreateSchemaStmt.Schemaname)
	case *pg.Node_DefineStmt:
		c.convertDefineStmt(n.DefineStmt)
	}
}

// CREATE TABLE

func (c *sqlConverter) convertCreateTable(stmt *pg.CreateStmt) {
	schema := c.getSchema(stmt.Relation.Schemaname)
	tbl := pgd.Table{
		Name: stmt.Relation.Relname,
	}

	// tablespace (#7)
	if stmt.Tablespacename != "" {
		tbl.Tablespace = stmt.Tablespacename
	}

	// inherits (#12)
	if len(stmt.InhRelations) > 0 {
		var parents []string
		for _, inh := range stmt.InhRelations {
			if rv, ok := inh.Node.(*pg.Node_RangeVar); ok {
				name := rv.RangeVar.Relname
				if rv.RangeVar.Schemaname != "" {
					name = rv.RangeVar.Schemaname + "." + name
				}
				parents = append(parents, name)
			}
		}
		tbl.Inherits = strings.Join(parents, ",")
	}

	// partition by (#13)
	if stmt.Partspec != nil {
		pb := &pgd.PartitionBy{}
		switch stmt.Partspec.Strategy {
		case pg.PartitionStrategy_PARTITION_STRATEGY_RANGE:
			pb.Type = "range"
		case pg.PartitionStrategy_PARTITION_STRATEGY_LIST:
			pb.Type = "list"
		case pg.PartitionStrategy_PARTITION_STRATEGY_HASH:
			pb.Type = "hash"
		}
		for _, p := range stmt.Partspec.PartParams {
			if pe, ok := p.Node.(*pg.Node_PartitionElem); ok {
				if pe.PartitionElem.Name != "" {
					pb.Columns = append(pb.Columns, pgd.ColRef{Name: pe.PartitionElem.Name})
				}
			}
		}
		tbl.PartitionBy = pb
	}

	// storage params — WITH (fillfactor=90, ...) (#10)
	if len(stmt.Options) > 0 {
		var params []pgd.WithParam
		for _, opt := range stmt.Options {
			if de, ok := opt.Node.(*pg.Node_DefElem); ok { //nolint:nestif
				val := defElemString(de.DefElem)
				if val == "" {
					val = strconv.FormatInt(defElemInt64(de.DefElem), 10)
				}
				params = append(params, pgd.WithParam{Name: de.DefElem.Defname, Value: val})
			}
		}
		if len(params) > 0 {
			tbl.With = &pgd.With{Params: params}
		}
	}

	for _, elt := range stmt.TableElts {
		switch n := elt.Node.(type) {
		case *pg.Node_ColumnDef:
			tbl.Columns = append(tbl.Columns, convertColumnDef(n.ColumnDef))
			extractColumnConstraints(&tbl, n.ColumnDef)
		case *pg.Node_Constraint:
			convertTableConstraint(&tbl, n.Constraint)
		}
	}

	schema.Tables = append(schema.Tables, tbl)
}

// extractColumnConstraints handles column-level constraints that affect the table (PK, UNIQUE, FK, CHECK).
func extractColumnConstraints(tbl *pgd.Table, col *pg.ColumnDef) {
	for _, conNode := range col.Constraints {
		con, ok := conNode.Node.(*pg.Node_Constraint)
		if !ok {
			continue
		}
		switch con.Constraint.Contype {
		case pg.ConstrType_CONSTR_PRIMARY:
			tbl.PK = &pgd.PrimaryKey{
				Name:    con.Constraint.Conname,
				Columns: []pgd.ColRef{{Name: col.Colname}},
			}
		case pg.ConstrType_CONSTR_UNIQUE:
			u := pgd.Unique{
				Name:    con.Constraint.Conname,
				Columns: []pgd.ColRef{{Name: col.Colname}},
			}
			if con.Constraint.NullsNotDistinct {
				u.NullsDistinct = "false"
			}
			tbl.Uniques = append(tbl.Uniques, u)
		case pg.ConstrType_CONSTR_FOREIGN:
			fk := convertFKConstraint(con.Constraint)
			if len(fk.Columns) == 0 {
				fk.Columns = []pgd.FKCol{{Name: col.Colname, References: col.Colname}}
			}
			tbl.FKs = append(tbl.FKs, fk)
		case pg.ConstrType_CONSTR_CHECK:
			ch := pgd.Check{Name: con.Constraint.Conname}
			if con.Constraint.RawExpr != nil {
				ch.Expression = nodeToSQL(con.Constraint.RawExpr)
			}
			tbl.Checks = append(tbl.Checks, ch)
		}
	}
}

func convertColumnDef(col *pg.ColumnDef) pgd.Column {
	typStr := typeFromNode(col.TypeName)

	c := pgd.Column{
		Name: col.Colname,
		Type: typStr,
	}

	// length/precision from type modifiers
	if col.TypeName != nil {
		applyTypeMods(&c, col.TypeName)
	}

	// storage & compression (#11)
	if col.StorageName != "" {
		c.Storage = col.StorageName
	}
	if col.Compression != "" {
		c.Compression = col.Compression
	}

	// column constraints
	for _, conNode := range col.Constraints {
		con, ok := conNode.Node.(*pg.Node_Constraint)
		if !ok {
			continue
		}
		switch con.Constraint.Contype {
		case pg.ConstrType_CONSTR_NOTNULL:
			c.Nullable = "false"
		case pg.ConstrType_CONSTR_DEFAULT:
			if con.Constraint.RawExpr != nil {
				c.Default = stripDefaultTypeCast(nodeToSQL(con.Constraint.RawExpr), c.Type)
			}
		case pg.ConstrType_CONSTR_IDENTITY:
			gen := "by-default"
			if con.Constraint.GeneratedWhen == "a" {
				gen = "always"
			}
			c.Identity = &pgd.Identity{Generated: gen}
		}
	}

	return c
}

func typeFromNode(tn *pg.TypeName) string {
	if tn == nil {
		return "unknown"
	}
	var parts []string
	for _, n := range tn.Names {
		if s, ok := n.Node.(*pg.Node_String_); ok {
			if s.String_.Sval == "pg_catalog" {
				continue
			}
			parts = append(parts, s.String_.Sval)
		}
	}

	var name string
	if len(parts) >= 2 && parts[0] != "public" {
		// schema-qualified type (e.g. myschema.custom_type)
		name = parts[0] + "." + parts[len(parts)-1]
	} else {
		name = parts[len(parts)-1]
	}

	if len(tn.ArrayBounds) > 0 {
		name += "[]"
	}

	return pgd.NormalizeType(name)
}

func applyTypeMods(c *pgd.Column, tn *pg.TypeName) {
	if len(tn.Typmods) == 0 {
		return
	}

	typeLower := strings.TrimSuffix(strings.ToLower(c.Type), "[]")

	switch {
	case pgd.NeedsLength(typeLower):
		// varchar(N), char(N), bit(N)
		// PG parser stores length as typmod + 4 offset for varchar/char
		if v := intFromNode(tn.Typmods[0]); v > 0 {
			c.Length = v
		}
	case typeLower == "numeric" || typeLower == "decimal":
		if v := intFromNode(tn.Typmods[0]); v > 0 {
			c.Precision = v
		}
		if len(tn.Typmods) >= 2 {
			if v := intFromNode(tn.Typmods[1]); v >= 0 {
				c.Scale = v
			}
		}
	}
}

func convertTableConstraint(tbl *pgd.Table, con *pg.Constraint) {
	switch con.Contype {
	case pg.ConstrType_CONSTR_PRIMARY:
		pk := pgd.PrimaryKey{Name: con.Conname}
		for _, k := range con.Keys {
			if s, ok := k.Node.(*pg.Node_String_); ok {
				pk.Columns = append(pk.Columns, pgd.ColRef{Name: s.String_.Sval})
			}
		}
		tbl.PK = &pk

	case pg.ConstrType_CONSTR_UNIQUE:
		u := pgd.Unique{Name: con.Conname}
		if con.NullsNotDistinct {
			u.NullsDistinct = "false"
		}
		for _, k := range con.Keys {
			if s, ok := k.Node.(*pg.Node_String_); ok {
				u.Columns = append(u.Columns, pgd.ColRef{Name: s.String_.Sval})
			}
		}
		tbl.Uniques = append(tbl.Uniques, u)

	case pg.ConstrType_CONSTR_CHECK:
		expr := ""
		if con.RawExpr != nil {
			expr = nodeToSQL(con.RawExpr)
		}
		tbl.Checks = append(tbl.Checks, pgd.Check{Name: con.Conname, Expression: expr})

	case pg.ConstrType_CONSTR_FOREIGN:
		fk := convertFKConstraint(con)
		tbl.FKs = append(tbl.FKs, fk)

	case pg.ConstrType_CONSTR_EXCLUSION:
		tbl.Excludes = append(tbl.Excludes, convertExcludeConstraint(con))
	}
}

// convertExcludeConstraint converts a pg EXCLUDE constraint to pgd.Exclude.
func convertExcludeConstraint(con *pg.Constraint) pgd.Exclude {
	ex := pgd.Exclude{
		Name:  con.Conname,
		Using: con.AccessMethod,
	}
	for _, node := range con.Exclusions {
		list := node.GetList()
		if list == nil || len(list.Items) < 2 {
			continue
		}
		var elem pgd.ExcludeElement
		// first item: IndexElem (column or expression)
		if ie := list.Items[0].GetIndexElem(); ie != nil {
			if ie.Name != "" {
				elem.Column = ie.Name
			} else if ie.Expr != nil {
				elem.Expression = nodeToSQL(ie.Expr)
			}
		}
		// second item: operator name list
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
		ex.Where = &pgd.WhereClause{Value: stripLiteralTypeCasts(nodeToSQL(con.WhereClause))}
	}
	return ex
}

func parseWithParams(options []*pg.Node) []pgd.WithParam {
	var params []pgd.WithParam
	for _, opt := range options {
		de := opt.GetDefElem()
		if de == nil || de.Arg == nil {
			continue
		}
		var val string
		if s := de.Arg.GetString_(); s != nil {
			val = s.Sval
		} else if iv := de.Arg.GetInteger(); iv != nil {
			val = strconv.FormatInt(int64(iv.Ival), 10)
		}
		if val != "" {
			params = append(params, pgd.WithParam{Name: de.Defname, Value: val})
		}
	}
	return params
}

// CREATE INDEX

func (c *sqlConverter) convertCreateIndex(stmt *pg.IndexStmt) { //nolint:gocognit,nestif,cyclop // pg_query AST requires deep type assertions
	schema := c.getSchema(stmt.Relation.Schemaname)
	idx := pgd.Index{
		Name:  stmt.Idxname,
		Table: stmt.Relation.Relname,
	}
	if stmt.Unique {
		idx.Unique = "true"
	}
	if stmt.AccessMethod != "" && stmt.AccessMethod != "btree" {
		idx.Using = stmt.AccessMethod
	}

	for _, param := range stmt.IndexParams {
		if ie, ok := param.Node.(*pg.Node_IndexElem); ok { //nolint:nestif
			elem := ie.IndexElem
			colName := elem.Name
			// ColumnRef with single field = column name (handles reserved words like "default")
			if colName == "" && elem.Expr != nil {
				if cr, ok := elem.Expr.Node.(*pg.Node_ColumnRef); ok && len(cr.ColumnRef.Fields) == 1 {
					if s, ok := cr.ColumnRef.Fields[0].Node.(*pg.Node_String_); ok {
						colName = s.String_.Sval
					}
				}
			}

			if colName != "" {
				ref := pgd.ColRef{Name: colName}
				if elem.Ordering == pg.SortByDir_SORTBY_DESC {
					ref.Order = "desc"
				}
				switch elem.NullsOrdering {
				case pg.SortByNulls_SORTBY_NULLS_FIRST:
					ref.Nulls = "first"
				case pg.SortByNulls_SORTBY_NULLS_LAST:
					ref.Nulls = "last"
				}
				idx.Columns = append(idx.Columns, ref)
			} else if elem.Expr != nil {
				idx.Expressions = append(idx.Expressions, pgd.Expression{Value: nodeToSQL(elem.Expr)})
			}
		}
	}

	if params := parseWithParams(stmt.Options); len(params) > 0 {
		idx.With = &pgd.With{Params: params}
	}

	if stmt.WhereClause != nil {
		idx.Where = &pgd.WhereClause{Value: stripLiteralTypeCasts(nodeToSQL(stmt.WhereClause))}
	}

	schema.Indexes = append(schema.Indexes, idx)
}

// ALTER TABLE ADD FOREIGN KEY

func (c *sqlConverter) convertAlterTable(stmt *pg.AlterTableStmt) { //nolint:gocognit,gocyclo,nestif,cyclop // ALTER TABLE has many subtype branches
	schema := c.getSchema(stmt.Relation.Schemaname)
	tableName := stmt.Relation.Relname

	tableIdx := -1
	for i, t := range schema.Tables {
		if t.Name == tableName {
			tableIdx = i
			break
		}
	}
	if tableIdx < 0 {
		return
	}

	for _, cmd := range stmt.Cmds {
		ac, ok := cmd.Node.(*pg.Node_AlterTableCmd)
		if !ok {
			continue
		}

		// ALTER TABLE ... ATTACH PARTITION child FOR VALUES ...
		if ac.AlterTableCmd.Subtype == pg.AlterTableType_AT_AttachPartition {
			attachPartitionChild(schema, tableIdx, ac.AlterTableCmd)
			continue
		}

		// ALTER TABLE ... ALTER COLUMN ... ADD GENERATED AS IDENTITY
		if ac.AlterTableCmd.Subtype == pg.AlterTableType_AT_AddIdentity { //nolint:nestif
			colName := ac.AlterTableCmd.Name
			if ac.AlterTableCmd.Def != nil {
				if con, isConstraint := ac.AlterTableCmd.Def.Node.(*pg.Node_Constraint); isConstraint {
					gen := "by-default"
					if con.Constraint.GeneratedWhen == "a" {
						gen = "always"
					}
					for k := range schema.Tables[tableIdx].Columns {
						if schema.Tables[tableIdx].Columns[k].Name == colName {
							schema.Tables[tableIdx].Columns[k].Identity = &pgd.Identity{Generated: gen}
							break
						}
					}
				}
			}
			continue
		}

		if ac.AlterTableCmd.Subtype != pg.AlterTableType_AT_AddConstraint {
			continue
		}
		conNode := ac.AlterTableCmd.Def
		if conNode == nil {
			continue
		}
		con, ok := conNode.Node.(*pg.Node_Constraint)
		if !ok {
			continue
		}
		switch con.Constraint.Contype {
		case pg.ConstrType_CONSTR_FOREIGN:
			fk := convertFKConstraint(con.Constraint)
			schema.Tables[tableIdx].FKs = append(schema.Tables[tableIdx].FKs, fk)
		case pg.ConstrType_CONSTR_PRIMARY:
			pk := pgd.PrimaryKey{Name: con.Constraint.Conname}
			for _, k := range con.Constraint.Keys {
				if s, ok := k.Node.(*pg.Node_String_); ok {
					pk.Columns = append(pk.Columns, pgd.ColRef{Name: s.String_.Sval})
				}
			}
			schema.Tables[tableIdx].PK = &pk
		case pg.ConstrType_CONSTR_UNIQUE:
			u := pgd.Unique{Name: con.Constraint.Conname}
			if con.Constraint.NullsNotDistinct {
				u.NullsDistinct = "false"
			}
			for _, k := range con.Constraint.Keys {
				if s, ok := k.Node.(*pg.Node_String_); ok {
					u.Columns = append(u.Columns, pgd.ColRef{Name: s.String_.Sval})
				}
			}
			schema.Tables[tableIdx].Uniques = append(schema.Tables[tableIdx].Uniques, u)
		case pg.ConstrType_CONSTR_CHECK:
			ch := pgd.Check{Name: con.Constraint.Conname}
			if con.Constraint.RawExpr != nil {
				ch.Expression = nodeToSQL(con.Constraint.RawExpr)
			}
			schema.Tables[tableIdx].Checks = append(schema.Tables[tableIdx].Checks, ch)
		case pg.ConstrType_CONSTR_EXCLUSION:
			schema.Tables[tableIdx].Excludes = append(schema.Tables[tableIdx].Excludes, convertExcludeConstraint(con.Constraint))
		}
	}
}

// CREATE VIEW

func (c *sqlConverter) convertView(stmt *pg.ViewStmt) {
	v := pgd.View{
		Name:   stmt.View.Relname,
		Schema: stmt.View.Schemaname,
	}
	if stmt.Query != nil {
		v.Query = stmtToSQL(stmt.Query)
	}
	c.views = append(c.views, v)
}

// CREATE MATERIALIZED VIEW

func (c *sqlConverter) convertMatView(stmt *pg.CreateTableAsStmt) {
	if stmt.Objtype != pg.ObjectType_OBJECT_MATVIEW {
		return
	}
	mv := pgd.MaterializedView{
		Name:   stmt.Into.Rel.Relname,
		Schema: stmt.Into.Rel.Schemaname,
	}
	if stmt.Query != nil {
		mv.Query = stmtToSQL(stmt.Query)
	}
	c.matViews = append(c.matViews, mv)
}

// CREATE FUNCTION

func (c *sqlConverter) convertFunction(stmt *pg.CreateFunctionStmt) { //nolint:gocognit,nestif,cyclop // pg_query AST requires deep type assertions
	qn := parseQualName(stmt.Funcname)
	f := pgd.Function{
		Name:   qn.name,
		Schema: qn.schema,
	}

	if stmt.IsProcedure {
		f.Kind = "procedure"
	}

	// return type
	if stmt.ReturnType != nil {
		ret := typeFromNode(stmt.ReturnType)
		if stmt.ReturnType.Setof {
			ret = "SETOF " + ret
		}
		f.Returns = ret
	}

	// arguments
	for _, p := range stmt.Parameters {
		if fp, ok := p.Node.(*pg.Node_FunctionParameter); ok {
			arg := pgd.FuncArg{
				Name: fp.FunctionParameter.Name,
				Type: typeFromNode(fp.FunctionParameter.ArgType),
			}
			switch fp.FunctionParameter.Mode {
			case pg.FunctionParameterMode_FUNC_PARAM_OUT:
				arg.Mode = "out"
			case pg.FunctionParameterMode_FUNC_PARAM_INOUT:
				arg.Mode = "inout"
			case pg.FunctionParameterMode_FUNC_PARAM_VARIADIC:
				arg.Mode = "variadic"
			}
			f.Args = append(f.Args, arg)
		}
	}

	// options (language, volatility, etc.)
	for _, opt := range stmt.Options {
		if de, ok := opt.Node.(*pg.Node_DefElem); ok { //nolint:nestif
			switch de.DefElem.Defname {
			case "language":
				if s := defElemString(de.DefElem); s != "" {
					f.Language = s
				}
			case "volatility":
				if s := defElemString(de.DefElem); s != "" {
					f.Volatility = s
				}
			case "security":
				if defElemBool(de.DefElem) {
					f.Security = "definer"
				}
			case "strict":
				if defElemBool(de.DefElem) {
					f.Strict = "true"
				}
			case "parallel":
				if s := defElemString(de.DefElem); s != "" {
					f.Parallel = s
				}
			case "as":
				// function body
				if de.DefElem.Arg != nil {
					if list := de.DefElem.Arg.GetList(); list != nil && len(list.Items) > 0 {
						if s := list.Items[0].GetString_(); s != nil {
							f.Body = strings.TrimSpace(s.Sval)
						}
					}
				}
			}
		}
	}

	c.functions = append(c.functions, f)
}

// CREATE SEQUENCE

func (c *sqlConverter) convertSequence(stmt *pg.CreateSeqStmt) {
	name := stmt.Sequence.Relname
	schema := stmt.Sequence.Schemaname

	// deduplicate: skip if sequence with same name already exists
	for _, existing := range c.sequences {
		if existing.Name == name && existing.Schema == schema {
			return
		}
	}

	seq := pgd.Sequence{
		Name:   name,
		Schema: schema,
	}
	// sequence options are in stmt.Options as DefElem
	for _, opt := range stmt.Options {
		if de, ok := opt.Node.(*pg.Node_DefElem); ok {
			switch de.DefElem.Defname {
			case "start":
				seq.Start = defElemInt64(de.DefElem)
			case "increment":
				seq.Increment = defElemInt64(de.DefElem)
			case "minvalue":
				seq.Min = defElemInt64(de.DefElem)
			case "maxvalue":
				seq.Max = defElemInt64(de.DefElem)
			case "cache":
				seq.Cache = defElemInt64(de.DefElem)
			case "cycle":
				if defElemBool(de.DefElem) {
					seq.Cycle = "true"
				}
			}
		}
	}
	c.sequences = append(c.sequences, seq)
}

// CREATE EXTENSION

func (c *sqlConverter) convertExtension(stmt *pg.CreateExtensionStmt) {
	ext := pgd.Extension{Name: stmt.Extname}
	for _, opt := range stmt.Options {
		if de, ok := opt.Node.(*pg.Node_DefElem); ok {
			if de.DefElem.Defname == "schema" {
				s := defElemString(de.DefElem)
				if s != "public" {
					ext.Schema = s
				}
			}
		}
	}
	c.extensions = append(c.extensions, ext)
}

// CREATE TYPE ... AS ENUM

func (c *sqlConverter) convertEnum(stmt *pg.CreateEnumStmt) {
	qn := parseQualName(stmt.TypeName)
	e := pgd.Enum{Name: qn.name, Schema: qn.schema}
	for _, v := range stmt.Vals {
		if s, ok := v.Node.(*pg.Node_String_); ok {
			e.Labels = append(e.Labels, s.String_.Sval)
		}
	}
	c.enums = append(c.enums, e)
}

// CREATE TYPE ... AS (composite)

func (c *sqlConverter) convertComposite(stmt *pg.CompositeTypeStmt) {
	ct := pgd.Composite{Name: stmt.Typevar.Relname}
	for _, col := range stmt.Coldeflist {
		if cd, ok := col.Node.(*pg.Node_ColumnDef); ok {
			ct.Fields = append(ct.Fields, pgd.CompositeField{
				Name: cd.ColumnDef.Colname,
				Type: typeFromNode(cd.ColumnDef.TypeName),
			})
		}
	}
	c.composites = append(c.composites, ct)
}

// CREATE DOMAIN

func (c *sqlConverter) convertDomain(stmt *pg.CreateDomainStmt) {
	qn := parseQualName(stmt.Domainname)
	d := pgd.Domain{
		Name:   qn.name,
		Schema: qn.schema,
		Type:   typeFromNode(stmt.TypeName),
	}
	for _, con := range stmt.Constraints {
		if cn, ok := con.Node.(*pg.Node_Constraint); ok {
			switch cn.Constraint.Contype {
			case pg.ConstrType_CONSTR_NOTNULL:
				d.NotNull = &struct{}{}
			case pg.ConstrType_CONSTR_CHECK:
				d.Constraints = append(d.Constraints, pgd.DomainConstraint{
					Name:       cn.Constraint.Conname,
					Expression: nodeToSQL(cn.Constraint.RawExpr),
				})
			case pg.ConstrType_CONSTR_DEFAULT:
				if cn.Constraint.RawExpr != nil {
					d.Default = nodeToSQL(cn.Constraint.RawExpr)
				}
			}
		}
	}
	c.domains = append(c.domains, d)
}

// CREATE TRIGGER

func (c *sqlConverter) convertTrigger(stmt *pg.CreateTrigStmt) {
	trig := pgd.Trigger{
		Name:  stmt.Trigname,
		Table: stmt.Relation.Relname,
	}

	// timing
	switch {
	case stmt.Timing&0x02 != 0:
		trig.Timing = "before"
	case stmt.Timing&0x04 != 0:
		trig.Timing = "instead-of"
	default:
		trig.Timing = "after"
	}

	// events
	var events []string
	if stmt.Events&0x04 != 0 {
		events = append(events, "insert")
	}
	if stmt.Events&0x08 != 0 {
		events = append(events, "delete")
	}
	if stmt.Events&0x10 != 0 {
		events = append(events, "update")
	}
	if stmt.Events&0x20 != 0 {
		events = append(events, "truncate")
	}
	trig.Events = strings.Join(events, ",")

	// for each
	if stmt.Row {
		trig.ForEach = "row"
	} else {
		trig.ForEach = "statement"
	}

	// function
	trig.Execute = pgd.TriggerExec{Function: funcName(stmt.Funcname)}

	c.triggers = append(c.triggers, trig)
}

// COMMENT ON (#5)

func (c *sqlConverter) convertComment(stmt *pg.CommentStmt) {
	cm := pgd.Comment{
		Value: stmt.Comment,
	}

	switch stmt.Objtype {
	case pg.ObjectType_OBJECT_TABLE:
		cm.On = "table"
	case pg.ObjectType_OBJECT_COLUMN:
		cm.On = "column"
	case pg.ObjectType_OBJECT_INDEX:
		cm.On = "index"
	case pg.ObjectType_OBJECT_FUNCTION:
		cm.On = "function"
	case pg.ObjectType_OBJECT_SCHEMA:
		cm.On = "schema"
	case pg.ObjectType_OBJECT_VIEW:
		cm.On = "view"
	case pg.ObjectType_OBJECT_MATVIEW:
		cm.On = "materialized-view"
	case pg.ObjectType_OBJECT_SEQUENCE:
		cm.On = "sequence"
	case pg.ObjectType_OBJECT_TYPE:
		cm.On = "type"
	case pg.ObjectType_OBJECT_DOMAIN:
		cm.On = "domain"
	case pg.ObjectType_OBJECT_TRIGGER:
		cm.On = "trigger"
	default:
		return
	}

	// extract object name from the Object node
	if stmt.Object != nil {
		switch obj := stmt.Object.Node.(type) {
		case *pg.Node_List:
			names := extractStrings(obj.List.Items)
			switch len(names) {
			case 1:
				cm.Name = names[0]
			case 2:
				if cm.On == "column" {
					cm.Table = names[0]
					cm.Name = names[1]
				} else {
					cm.Schema = names[0]
					cm.Name = names[1]
				}
			case 3:
				cm.Schema = names[0]
				cm.Table = names[1]
				cm.Name = names[2]
			}
		case *pg.Node_String_:
			cm.Name = obj.String_.Sval
		case *pg.Node_TypeName:
			names := extractStrings(obj.TypeName.Names)
			switch len(names) {
			case 1:
				cm.Name = names[0]
			case 2:
				cm.Schema = names[0]
				cm.Name = names[1]
			}
		case *pg.Node_ObjectWithArgs:
			names := extractStrings(obj.ObjectWithArgs.Objname)
			switch len(names) {
			case 1:
				cm.Name = names[0]
			case 2:
				cm.Schema = names[0]
				cm.Name = names[1]
			}
		}
	}

	c.comments = append(c.comments, cm)
}

// FK constraint helper

// CREATE AGGREGATE (DefineStmt with kind=OBJECT_AGGREGATE)

func (c *sqlConverter) convertDefineStmt(stmt *pg.DefineStmt) {
	if stmt.Kind != pg.ObjectType_OBJECT_AGGREGATE {
		return
	}

	qn := parseQualName(stmt.Defnames)
	f := pgd.Function{
		Name:   qn.name,
		Schema: qn.schema,
		Kind:   "aggregate",
	}

	// arguments from Args[0] (List of FunctionParameter)
	if len(stmt.Args) > 0 {
		if list, ok := stmt.Args[0].Node.(*pg.Node_List); ok {
			for _, item := range list.List.Items {
				if fp, ok := item.Node.(*pg.Node_FunctionParameter); ok {
					f.Args = append(f.Args, pgd.FuncArg{
						Name: fp.FunctionParameter.Name,
						Type: typeFromNode(fp.FunctionParameter.ArgType),
					})
				}
			}
		}
	}

	// definition options (sfunc, stype, finalfunc, initcond, sortop, combinefunc)
	for _, opt := range stmt.Definition {
		de, ok := opt.Node.(*pg.Node_DefElem)
		if !ok {
			continue
		}
		val := ""
		if de.DefElem.Arg != nil {
			if tn, ok := de.DefElem.Arg.Node.(*pg.Node_TypeName); ok {
				val = typeFromNode(tn.TypeName)
			} else if s, ok := de.DefElem.Arg.Node.(*pg.Node_String_); ok {
				val = s.String_.Sval
			}
		}
		switch de.DefElem.Defname {
		case "sfunc":
			f.SFunc = val
		case "stype":
			f.SType = val
		case "finalfunc":
			f.FinalFunc = val
		case "initcond":
			f.InitCond = val
		case "sortop":
			f.SortOp = val
		case "combinefunc":
			f.CombineFunc = val
		}
	}

	c.functions = append(c.functions, f)
}

func convertFKConstraint(con *pg.Constraint) pgd.ForeignKey {
	fk := pgd.ForeignKey{
		Name:     con.Conname,
		OnDelete: pgd.FKActionFromPGCode(con.FkDelAction),
		OnUpdate: pgd.FKActionFromPGCode(con.FkUpdAction),
	}

	if con.Pktable != nil {
		fk.ToTable = con.Pktable.Relname
		// schema-qualified FK reference (#2)
		if con.Pktable.Schemaname != "" && con.Pktable.Schemaname != "public" {
			fk.ToTable = con.Pktable.Schemaname + "." + fk.ToTable
		}
	}

	fkCols := extractStrings(con.FkAttrs)
	pkCols := extractStrings(con.PkAttrs)

	// When REFERENCES table has no column list, PG implies the target PK.
	// Use FK column names as referenced column names (common convention: same name).
	if len(pkCols) == 0 {
		pkCols = fkCols
	}

	for i := 0; i < len(fkCols) && i < len(pkCols); i++ {
		fk.Columns = append(fk.Columns, pgd.FKCol{Name: fkCols[i], References: pkCols[i]})
	}

	if con.Deferrable {
		fk.Deferrable = "true"
	}
	if con.FkMatchtype == "f" {
		fk.Match = "full"
	}

	return fk
}

// helpers

func extractStrings(nodes []*pg.Node) []string {
	var out []string
	for _, n := range nodes {
		if s, ok := n.Node.(*pg.Node_String_); ok {
			out = append(out, s.String_.Sval)
		}
	}
	return out
}

func intFromNode(n *pg.Node) int {
	switch v := n.Node.(type) {
	case *pg.Node_Integer:
		return int(v.Integer.Ival)
	case *pg.Node_AConst:
		if v.AConst.Val != nil {
			if iv, ok := v.AConst.Val.(*pg.A_Const_Ival); ok {
				return int(iv.Ival.Ival)
			}
		}
	}
	return 0
}

// stmtToSQL deparses a statement node (e.g. SELECT) directly back to SQL.
func stmtToSQL(n *pg.Node) string {
	result, err := pgquery.Deparse(&pg.ParseResult{
		Stmts: []*pg.RawStmt{{Stmt: n}},
	})
	if err != nil {
		return ""
	}
	return cleanOperators(result)
}

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
	result = strings.TrimPrefix(result, "SELECT ")
	return cleanOperators(result)
}

// stripDefaultTypeCast removes redundant type casts from default values.
// pg_dump normalizes `DEFAULT ”` to `DEFAULT ”::text` — strip the cast when it matches the column type.
func stripDefaultTypeCast(def, colType string) string {
	idx := strings.LastIndex(def, "::")
	if idx <= 0 {
		return def
	}
	valuePart := def[:idx]
	if len(valuePart) == 0 {
		return def
	}
	if normalizeTypeName(colType) == normalizeTypeName(def[idx+2:]) {
		return valuePart
	}
	return def
}

// normalizeTypeName maps PG type aliases to canonical forms for comparison.
func normalizeTypeName(t string) string {
	t = strings.TrimSpace(strings.ToLower(t))
	// handle array suffix
	arraySuffix := ""
	if strings.HasSuffix(t, "[]") {
		arraySuffix = "[]"
		t = strings.TrimSuffix(t, "[]")
	}
	switch t {
	case "int", "int4", "integer":
		t = "integer"
	case "int8", "bigint":
		t = "bigint"
	case "int2", "smallint":
		t = "smallint"
	case "float4", "real":
		t = "real"
	case "float8", "double precision":
		t = "double precision"
	case "bool", "boolean":
		t = "boolean"
	case "varchar", "character varying":
		t = "varchar"
	case "char", "character":
		t = "char"
	case "timestamptz", "timestamp with time zone":
		t = "timestamptz"
	case "timestamp", "timestamp without time zone":
		t = "timestamp"
	case "timetz", "time with time zone":
		t = "timetz"
	case "time", "time without time zone":
		t = "time"
	}
	return t + arraySuffix
}

// stripLiteralTypeCasts removes type casts from string literals in SQL expressions.
// pg_dump adds explicit casts like ”::text or '{}'::jsonb — strip them for stable round-trip.
func stripLiteralTypeCasts(expr string) string {
	var b strings.Builder
	b.Grow(len(expr))
	i := 0
	for i < len(expr) {
		// find next single-quoted literal
		q := strings.Index(expr[i:], "'")
		if q < 0 {
			b.WriteString(expr[i:])
			break
		}
		// write everything before the quote
		b.WriteString(expr[i : i+q])
		// find closing quote (handle '' escapes)
		start := i + q
		j := start + 1
		for j < len(expr) {
			if expr[j] == '\'' {
				if j+1 < len(expr) && expr[j+1] == '\'' {
					j += 2 // escaped quote
					continue
				}
				break // closing quote
			}
			j++
		}
		if j >= len(expr) {
			b.WriteString(expr[start:])
			break
		}
		literal := expr[start : j+1] // 'value' including quotes
		rest := expr[j+1:]
		// check for ::type immediately after closing quote
		if strings.HasPrefix(rest, "::") {
			// find end of type name (letters, digits, _, [], spaces for "character varying" etc.)
			k := 2
			for k < len(rest) {
				ch := rest[k]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '[' || ch == ']' || ch == ' ' {
					k++
				} else {
					break
				}
			}
			typeName := strings.TrimRight(rest[2:k], " ")
			if typeName != "" {
				b.WriteString(literal)
				i = j + 1 + k
				continue
			}
		}
		b.WriteString(literal)
		i = j + 1
	}
	return b.String()
}

// cleanOperators removes schema-qualified operator syntax that pg_query deparse produces.
// "OPERATOR(public.->)" → "->", "OPERATOR(public.->>)" → "->>"
func cleanOperators(s string) string {
	for {
		idx := strings.Index(s, "OPERATOR(")
		if idx < 0 {
			return s
		}
		end := strings.Index(s[idx:], ")")
		if end < 0 {
			return s
		}
		// extract operator: OPERATOR(schema.op) → op
		inner := s[idx+9 : idx+end] // "public.->"
		if dot := strings.LastIndex(inner, "."); dot >= 0 {
			op := inner[dot+1:]
			s = s[:idx] + op + s[idx+end+1:]
		} else {
			s = s[:idx] + inner + s[idx+end+1:]
		}
	}
}

// cleanDump removes pg_dump noise that the SQL parser can't handle:
// backslash commands (\connect, \restrict), SET statements, comments, empty lines.
func cleanDump(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "--"):
			continue
		case strings.HasPrefix(trimmed, "\\"):
			continue
		case strings.HasPrefix(trimmed, "SET "):
			continue
		case strings.HasPrefix(trimmed, "SELECT pg_catalog."):
			continue
		default:
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type qualName struct {
	schema string
	name   string
}

func parseQualName(nodes []*pg.Node) qualName {
	var parts []string
	for _, n := range nodes {
		if s, ok := n.Node.(*pg.Node_String_); ok {
			if s.String_.Sval != "pg_catalog" {
				parts = append(parts, s.String_.Sval)
			}
		}
	}
	switch len(parts) {
	case 0:
		return qualName{}
	case 1:
		return qualName{name: parts[0]}
	default:
		schema := parts[0]
		if schema == "public" {
			schema = "" // default schema, omit
		}
		return qualName{schema: schema, name: parts[len(parts)-1]}
	}
}

func funcName(nodes []*pg.Node) string {
	return parseQualName(nodes).name
}

func defElemString(de *pg.DefElem) string {
	if de.Arg == nil {
		return ""
	}
	if s, ok := de.Arg.Node.(*pg.Node_String_); ok {
		return s.String_.Sval
	}
	return ""
}

func defElemInt64(de *pg.DefElem) int64 {
	if de.Arg == nil {
		return 0
	}
	if i, ok := de.Arg.Node.(*pg.Node_Integer); ok {
		return int64(i.Integer.Ival)
	}
	return 0
}

func defElemBool(de *pg.DefElem) bool {
	if de.Arg == nil {
		// defelem without arg often means TRUE
		return true
	}
	if i, ok := de.Arg.Node.(*pg.Node_Boolean); ok {
		return i.Boolean.Boolval
	}
	if i, ok := de.Arg.Node.(*pg.Node_Integer); ok {
		return i.Integer.Ival != 0
	}
	return false
}

func attachPartitionChild(schema *pgd.Schema, tableIdx int, cmd *pg.AlterTableCmd) {
	pcmd, ok := cmd.Def.Node.(*pg.Node_PartitionCmd)
	if !ok {
		return
	}
	childName := pcmd.PartitionCmd.Name.Relname
	parentName := schema.Tables[tableIdx].Name
	for k := range schema.Tables {
		if schema.Tables[k].Name != childName {
			continue
		}
		schema.Tables[k].PartitionOf = parentName
		if pcmd.PartitionCmd.Bound != nil {
			schema.Tables[k].PartitionBound = &pgd.PartitionBound{
				Value: deparseBound(pcmd.PartitionCmd.Bound),
			}
		}
		break
	}
}

func deparseBound(b *pg.PartitionBoundSpec) string {
	if b.IsDefault {
		return "DEFAULT"
	}
	deparseVals := func(nodes []*pg.Node) string {
		var vals []string
		for _, n := range nodes {
			vals = append(vals, deparseNode(n))
		}
		return strings.Join(vals, ", ")
	}
	if len(b.Lowerdatums) > 0 || len(b.Upperdatums) > 0 {
		return fmt.Sprintf("FOR VALUES FROM (%s) TO (%s)", deparseVals(b.Lowerdatums), deparseVals(b.Upperdatums))
	}
	if len(b.Listdatums) > 0 {
		return fmt.Sprintf("FOR VALUES IN (%s)", deparseVals(b.Listdatums))
	}
	if b.Modulus > 0 {
		return fmt.Sprintf("FOR VALUES WITH (MODULUS %d, REMAINDER %d)", b.Modulus, b.Remainder)
	}
	return ""
}

func deparseNode(n *pg.Node) string {
	switch v := n.Node.(type) {
	case *pg.Node_String_:
		return "'" + v.String_.Sval + "'"
	case *pg.Node_Integer:
		return strconv.FormatInt(int64(v.Integer.Ival), 10)
	case *pg.Node_Float:
		return v.Float.Fval
	case *pg.Node_AConst:
		return deparseAConst(v.AConst)
	case *pg.Node_ColumnRef:
		return deparseColumnRef(v.ColumnRef)
	}
	return "NULL"
}

func deparseAConst(ac *pg.A_Const) string {
	if ac.Val == nil {
		return "NULL"
	}
	switch cv := ac.Val.(type) {
	case *pg.A_Const_Sval:
		return "'" + cv.Sval.Sval + "'"
	case *pg.A_Const_Ival:
		return strconv.FormatInt(int64(cv.Ival.Ival), 10)
	case *pg.A_Const_Fval:
		return cv.Fval.Fval
	}
	return "NULL"
}

func deparseColumnRef(cr *pg.ColumnRef) string {
	if len(cr.Fields) == 0 {
		return "NULL"
	}
	s, ok := cr.Fields[0].Node.(*pg.Node_String_)
	if !ok {
		return "NULL"
	}
	switch s.String_.Sval {
	case "minvalue":
		return "MINVALUE"
	case "maxvalue":
		return "MAXVALUE"
	}
	return "NULL"
}

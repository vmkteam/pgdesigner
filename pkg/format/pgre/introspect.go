package pgre

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

type introspector struct {
	db   *pg.DB
	ctx  context.Context
	opts Options
}

// Raw row types for pg_catalog queries

type pgTable struct {
	OID            int
	SchemaName     string
	TableName      string
	RelKind        string
	RelPersistence string
	Tablespace     string
	Comment        string
}

type pgColumn struct {
	Name           string
	AttNum         int
	Type           string
	NotNull        bool
	DefaultValue   string
	AttIdentity    string
	AttGenerated   string
	AttCompression string
	AttStorage     string
	TypeStorage    string
	Comment        string
	Collation      string
}

type pgConstraint struct {
	Name          string
	ConType       string // p=PK, u=UNIQUE, c=CHECK, x=EXCLUDE, f=FK
	ConKey        []int  `pg:",array"`
	ConfRelID     int
	ConfKey       []int `pg:",array"`
	ConfDelType   string
	ConfUpdType   string
	ConfMatchType string
	Deferrable    bool
	Deferred      bool
	Def           string
	CheckExpr     string
}

type pgIndex struct {
	IndexName string
	IsUnique  bool
	Method    string
	Def       string
	Predicate string
}

type pgView struct {
	Name       string
	SchemaName string
	Definition string
	RelKind    string // v=view, m=materialized view
}

type pgFunction struct {
	Name       string
	SchemaName string
	Definition string
	Language   string
	Volatile   string
	IsStrict   bool
	SecDef     bool
	Parallel   string
	Kind       string // f=function, p=procedure
}

type pgTrigger struct {
	Name       string
	TableName  string
	SchemaName string
	Definition string
}

type pgExtension struct {
	Name       string
	SchemaName string
	Version    string
}

type pgSequence struct {
	Name      string
	Schema    string
	Start     int64
	Increment int64
	Min       int64
	Max       int64
	Cache     int64
	Cycle     bool
}

type pgEnum struct {
	Name   string
	Schema string
	Labels []string
}

type pgDomain struct {
	Name     string
	Schema   string
	BaseType string
	NotNull  bool
	Default  string
}

type pgPartitionParent struct {
	TableName    string
	PartStrategy string
	PartKeyDef   string
}

type pgPartitionChild struct {
	ChildName  string
	ParentName string
	BoundExpr  string
}

func (i *introspector) introspect() (*pgd.Project, error) {
	// Get database name and PG version
	var dbName, pgVersion string
	_, _ = i.db.QueryOne(pg.Scan(&dbName), "SELECT current_database()")
	_, _ = i.db.QueryOne(pg.Scan(&pgVersion), "SELECT split_part(version(), ' ', 2)")

	// Get major version
	if parts := strings.Split(pgVersion, "."); len(parts) > 0 {
		pgVersion = parts[0]
	}

	// Introspect tables
	tables, err := i.queryTables()
	if err != nil {
		return nil, fmt.Errorf("querying tables: %w", err)
	}

	// Build schemas
	schemaMap := make(map[string]*pgd.Schema)
	for _, t := range tables {
		s, ok := schemaMap[t.SchemaName]
		if !ok {
			s = &pgd.Schema{Name: t.SchemaName}
			schemaMap[t.SchemaName] = s
		}

		table, indexes, err := i.convertTable(t) //nolint:govet // err shadow is intentional
		if err != nil {
			return nil, fmt.Errorf("converting table %s.%s: %w", t.SchemaName, t.TableName, err)
		}

		s.Tables = append(s.Tables, *table)
		s.Indexes = append(s.Indexes, indexes...)
	}

	// Build schemas slice
	var schemas []pgd.Schema
	for _, s := range schemaMap {
		schemas = append(schemas, *s)
	}

	project := &pgd.Project{
		Version:       1,
		PgVersion:     pgVersion,
		DefaultSchema: "public",
		ProjectMeta: pgd.ProjectMeta{
			Name: dbName,
			Settings: pgd.Settings{
				Naming:   pgd.Naming{Convention: "camelCase"},
				Defaults: pgd.Defaults{Nullable: "true", OnDelete: "no action", OnUpdate: "no action"},
			},
		},
		Schemas: schemas,
	}

	// Full mode: views, functions, triggers, extensions, domains, enums
	if i.opts.Full {
		if intErr := i.introspectFull(project); intErr != nil { //nolint:govet
			return nil, err
		}
	}

	// Sequences (always in min)
	seqs, err := i.querySequences()
	if err == nil {
		for _, s := range seqs {
			project.Sequences = append(project.Sequences, convertSequence(s))
		}
	}

	// Partitions: attach partition info to tables
	i.introspectPartitions(project)

	return project, nil
}

func (i *introspector) schemaFilter() string {
	if len(i.opts.Schemas) == 0 {
		return defaultSchemaFilter
	}
	quoted := make([]string, len(i.opts.Schemas))
	for j, s := range i.opts.Schemas {
		quoted[j] = "'" + s + "'"
	}
	return "AND n.nspname IN (" + strings.Join(quoted, ",") + ")"
}

func (i *introspector) queryTables() ([]pgTable, error) {
	var tables []pgTable
	_, err := i.db.Query(&tables, fmt.Sprintf(`
		SELECT c.oid, n.nspname AS schema_name, c.relname AS table_name,
		       c.relkind AS rel_kind, c.relpersistence AS rel_persistence,
		       COALESCE(t.spcname, '') AS tablespace,
		       COALESCE(obj_description(c.oid), '') AS comment
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_tablespace t ON t.oid = c.reltablespace
		WHERE c.relkind IN ('r', 'p')
		  %s
		ORDER BY n.nspname, c.relname
	`, i.schemaFilter()))
	return tables, err
}

func (i *introspector) queryColumns(tableOID int) ([]pgColumn, error) {
	var cols []pgColumn
	_, err := i.db.Query(&cols, `
		SELECT a.attname AS name, a.attnum AS att_num,
		       pg_catalog.format_type(a.atttypid, a.atttypmod) AS type,
		       a.attnotnull AS not_null,
		       COALESCE(pg_get_expr(d.adbin, d.adrelid), '') AS default_value,
		       COALESCE(a.attidentity::text, '') AS att_identity,
		       COALESCE(a.attgenerated::text, '') AS att_generated,
		       COALESCE(a.attcompression::text, '') AS att_compression,
		       a.attstorage::text AS att_storage,
		       t.typstorage::text AS type_storage,
		       COALESCE(col_description(a.attrelid, a.attnum), '') AS comment,
		       COALESCE(coll.collname, '') AS collation
		FROM pg_attribute a
		JOIN pg_type t ON t.oid = a.atttypid
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		LEFT JOIN pg_collation coll ON coll.oid = a.attcollation AND a.attcollation <> 0
		WHERE a.attrelid = ?
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`, tableOID)
	return cols, err
}

func (i *introspector) queryConstraints(tableOID int) ([]pgConstraint, error) {
	var cons []pgConstraint
	_, err := i.db.Query(&cons, `
		SELECT c.conname AS name, c.contype AS con_type,
		       c.conkey AS con_key,
		       COALESCE(c.confrelid, 0) AS conf_rel_id,
		       c.confkey AS conf_key,
		       COALESCE(c.confdeltype::text, '') AS conf_del_type,
		       COALESCE(c.confupdtype::text, '') AS conf_upd_type,
		       COALESCE(c.confmatchtype::text, '') AS conf_match_type,
		       c.condeferrable AS deferrable,
		       c.condeferred AS deferred,
		       pg_get_constraintdef(c.oid) AS def,
		       COALESCE(pg_get_expr(c.conbin, c.conrelid), '') AS check_expr
		FROM pg_constraint c
		WHERE c.conrelid = ?
		ORDER BY c.contype, c.conname
	`, tableOID)
	return cons, err
}

func (i *introspector) queryIndexes(tableOID int) ([]pgIndex, error) {
	var idxs []pgIndex
	_, err := i.db.Query(&idxs, `
		SELECT ic.relname AS index_name,
		       i.indisunique AS is_unique,
		       am.amname AS method,
		       pg_get_indexdef(i.indexrelid) AS def,
		       COALESCE(pg_get_expr(i.indpred, i.indrelid), '') AS predicate
		FROM pg_index i
		JOIN pg_class ic ON ic.oid = i.indexrelid
		JOIN pg_am am ON am.oid = ic.relam
		WHERE i.indrelid = ?
		  AND NOT i.indisprimary
		  AND NOT EXISTS (
		    SELECT 1 FROM pg_constraint c
		    WHERE c.conindid = i.indexrelid
		  )
		ORDER BY ic.relname
	`, tableOID)
	return idxs, err
}

func (i *introspector) queryRefTableName(oid int) (schema, name string) {
	_, _ = i.db.QueryOne(pg.Scan(&schema, &name), `
		SELECT n.nspname, c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.oid = ?
	`, oid)
	return
}

func (i *introspector) queryColumnNames(tableOID int, attNums []int) []string {
	if len(attNums) == 0 {
		return nil
	}
	var names []string
	_, _ = i.db.Query(pg.Scan(pg.Array(&names)), `
		SELECT array_agg(a.attname ORDER BY ord)
		FROM unnest(?::int[]) WITH ORDINALITY AS u(attnum, ord)
		JOIN pg_attribute a ON a.attrelid = ? AND a.attnum = u.attnum
	`, pg.Array(attNums), tableOID)
	return names
}

func (i *introspector) querySequences() ([]pgSequence, error) {
	var seqs []pgSequence
	_, err := i.db.Query(&seqs, fmt.Sprintf(`
		SELECT s.relname AS name, n.nspname AS schema,
		       seq.seqstart AS start, seq.seqincrement AS increment,
		       seq.seqmin AS min, seq.seqmax AS max,
		       seq.seqcache AS cache, seq.seqcycle AS cycle
		FROM pg_class s
		JOIN pg_namespace n ON n.oid = s.relnamespace
		JOIN pg_sequence seq ON seq.seqrelid = s.oid
		WHERE s.relkind = 'S'
		  %s
		  AND NOT EXISTS (
		    SELECT 1 FROM pg_depend d
		    WHERE d.objid = s.oid AND d.deptype = 'i'
		  )
		ORDER BY n.nspname, s.relname
	`, i.schemaFilter()))
	return seqs, err
}

// Full mode queries

func (i *introspector) introspectFull(project *pgd.Project) error { //nolint:gocognit,govet,unparam
	// Extensions
	exts, err := i.queryExtensions()
	if err == nil {
		for _, e := range exts {
			project.Extensions = append(project.Extensions, pgd.Extension{
				Name:    e.Name,
				Schema:  e.SchemaName,
				Version: e.Version,
			})
		}
	}

	// Enums
	enums, err := i.queryEnums()
	if err == nil && len(enums) > 0 {
		if project.Types == nil {
			project.Types = &pgd.Types{}
		}
		for _, e := range enums {
			project.Types.Enums = append(project.Types.Enums, pgd.Enum{
				Name:   e.Name,
				Schema: e.Schema,
				Labels: e.Labels,
			})
		}
	}

	// Domains
	domains, err := i.queryDomains()
	if err == nil && len(domains) > 0 {
		if project.Types == nil {
			project.Types = &pgd.Types{}
		}
		for _, d := range domains {
			dom := pgd.Domain{Name: d.Name, Schema: d.Schema, Type: d.BaseType}
			if d.NotNull {
				dom.NotNull = &struct{}{}
			}
			if d.Default != "" {
				dom.Default = d.Default
			}
			project.Types.Domains = append(project.Types.Domains, dom)
		}
	}

	// Views
	views, err := i.queryViews()
	if err == nil && len(views) > 0 {
		if project.Views == nil {
			project.Views = &pgd.Views{}
		}
		for _, v := range views {
			if v.RelKind == "v" {
				project.Views.Views = append(project.Views.Views, pgd.View{
					Name:   v.Name,
					Schema: v.SchemaName,
					Query:  v.Definition,
				})
			} else {
				project.Views.MatViews = append(project.Views.MatViews, pgd.MaterializedView{
					Name:   v.Name,
					Schema: v.SchemaName,
					Query:  v.Definition,
				})
			}
		}
	}

	// Materialized view indexes
	i.introspectMatViewIndexes(project)

	// Functions
	funcs, err := i.queryFunctions()
	if err == nil {
		for _, f := range funcs {
			fn := pgd.Function{
				Name:     f.Name,
				Schema:   f.SchemaName,
				Language: f.Language,
			}
			if f.Kind == "p" {
				fn.Kind = "procedure"
			}
			switch f.Volatile {
			case "i":
				fn.Volatility = "immutable"
			case "s":
				fn.Volatility = "stable"
			}
			if f.SecDef {
				fn.Security = "definer"
			}
			switch f.Parallel {
			case "s":
				fn.Parallel = "safe"
			case "r":
				fn.Parallel = "restricted"
			}
			if f.IsStrict {
				fn.Strict = "true"
			}
			// Body from pg_get_functiondef — contains full CREATE FUNCTION
			fn.Body = f.Definition
			project.Functions = append(project.Functions, fn)
		}
	}

	// Triggers
	triggers, err := i.queryTriggers()
	if err == nil {
		for _, t := range triggers {
			project.Triggers = append(project.Triggers, pgd.Trigger{
				Name:  t.Name,
				Table: t.TableName,
			})
		}
	}

	return nil
}

func (i *introspector) queryExtensions() ([]pgExtension, error) {
	var exts []pgExtension
	_, err := i.db.Query(&exts, `
		SELECT e.extname AS name, n.nspname AS schema_name, e.extversion AS version
		FROM pg_extension e
		JOIN pg_namespace n ON n.oid = e.extnamespace
		WHERE e.extname != 'plpgsql'
	`)
	return exts, err
}

func (i *introspector) queryEnums() ([]pgEnum, error) {
	var enums []pgEnum
	_, err := i.db.Query(&enums, fmt.Sprintf(`
		SELECT t.typname AS name, n.nspname AS schema,
		       array_agg(e.enumlabel ORDER BY e.enumsortorder) AS labels
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		JOIN pg_enum e ON e.enumtypid = t.oid
		WHERE 1=1 %s
		GROUP BY t.typname, n.nspname
		ORDER BY n.nspname, t.typname
	`, i.schemaFilter()))
	return enums, err
}

func (i *introspector) queryDomains() ([]pgDomain, error) {
	var domains []pgDomain
	_, err := i.db.Query(&domains, fmt.Sprintf(`
		SELECT t.typname AS name, n.nspname AS schema,
		       pg_catalog.format_type(t.typbasetype, t.typtypmod) AS base_type,
		       t.typnotnull AS not_null,
		       COALESCE(t.typdefault, '') AS default
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typtype = 'd'
		  %s
		ORDER BY n.nspname, t.typname
	`, i.schemaFilter()))
	return domains, err
}

func (i *introspector) queryViews() ([]pgView, error) {
	var views []pgView
	_, err := i.db.Query(&views, fmt.Sprintf(`
		SELECT c.relname AS name, n.nspname AS schema_name,
		       pg_get_viewdef(c.oid, true) AS definition,
		       c.relkind AS rel_kind
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('v', 'm')
		  %s
		ORDER BY n.nspname, c.relname
	`, i.schemaFilter()))
	return views, err
}

func (i *introspector) queryFunctions() ([]pgFunction, error) {
	var funcs []pgFunction
	_, err := i.db.Query(&funcs, fmt.Sprintf(`
		SELECT p.proname AS name, n.nspname AS schema_name,
		       pg_get_functiondef(p.oid) AS definition,
		       l.lanname AS language,
		       p.provolatile AS volatile,
		       p.proisstrict AS is_strict,
		       p.prosecdef AS sec_def,
		       p.proparallel AS parallel,
		       p.prokind AS kind
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_language l ON l.oid = p.prolang
		WHERE p.prokind IN ('f', 'p')
		  %s
		ORDER BY n.nspname, p.proname
	`, i.schemaFilter()))
	return funcs, err
}

func (i *introspector) queryPartitionParents() ([]pgPartitionParent, error) {
	var parents []pgPartitionParent
	_, err := i.db.Query(&parents, fmt.Sprintf(`
		SELECT c.relname AS table_name,
		       CASE pt.partstrat WHEN 'r' THEN 'range' WHEN 'l' THEN 'list' WHEN 'h' THEN 'hash' END AS part_strategy,
		       pg_get_partkeydef(c.oid) AS part_key_def
		FROM pg_class c
		JOIN pg_partitioned_table pt ON c.oid = pt.partrelid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE 1=1 %s
		ORDER BY c.relname
	`, i.schemaFilter()))
	return parents, err
}

func (i *introspector) queryPartitionChildren() ([]pgPartitionChild, error) {
	var children []pgPartitionChild
	_, err := i.db.Query(&children, fmt.Sprintf(`
		SELECT child.relname AS child_name, parent.relname AS parent_name,
		       pg_get_expr(child.relpartbound, child.oid) AS bound_expr
		FROM pg_class child
		JOIN pg_inherits inh ON child.oid = inh.inhrelid
		JOIN pg_class parent ON inh.inhparent = parent.oid
		JOIN pg_namespace n ON child.relnamespace = n.oid
		WHERE child.relispartition
		  %s
		ORDER BY parent.relname, child.relname
	`, i.schemaFilter()))
	return children, err
}

func (i *introspector) introspectPartitions(project *pgd.Project) {
	parents, err := i.queryPartitionParents()
	if err != nil || len(parents) == 0 {
		return
	}

	// Set PartitionBy on parent tables
	parentSet := make(map[string]bool)
	for _, pp := range parents {
		parentSet[pp.TableName] = true
		for si := range project.Schemas {
			for ti := range project.Schemas[si].Tables {
				if project.Schemas[si].Tables[ti].Name == pp.TableName {
					cols := parsePartKeyDef(pp.PartKeyDef)
					project.Schemas[si].Tables[ti].PartitionBy = &pgd.PartitionBy{
						Type:    pp.PartStrategy,
						Columns: cols,
					}
				}
			}
		}
	}

	children, err := i.queryPartitionChildren()
	if err != nil || len(children) == 0 {
		return
	}

	// Group children by parent
	childMap := make(map[string][]pgPartitionChild)
	for _, ch := range children {
		childMap[ch.ParentName] = append(childMap[ch.ParentName], ch)
	}

	// Attach children and remove child tables from schema
	for si := range project.Schemas {
		s := &project.Schemas[si]
		for ti := range s.Tables {
			t := &s.Tables[ti]
			if chs, ok := childMap[t.Name]; ok {
				for _, ch := range chs {
					t.Partitions = append(t.Partitions, pgd.Partition{
						Name:  ch.ChildName,
						Bound: ch.BoundExpr,
					})
				}
			}
		}

		// Remove child partition tables (they are now nested inside parent)
		childNames := make(map[string]bool)
		for _, chs := range childMap {
			for _, ch := range chs {
				childNames[ch.ChildName] = true
			}
		}
		var filtered []pgd.Table
		for _, t := range s.Tables {
			if !childNames[t.Name] {
				filtered = append(filtered, t)
			}
		}
		s.Tables = filtered
	}
}

// parsePartKeyDef parses pg_get_partkeydef output like "RANGE (payment_date)" or "LIST (country, city)".
// It strips the strategy keyword and parentheses, returning just column names.
func parsePartKeyDef(def string) []pgd.ColRef {
	// Strip strategy prefix: "RANGE (cols)" → "cols"
	if idx := strings.IndexByte(def, '('); idx >= 0 {
		end := strings.LastIndexByte(def, ')')
		if end > idx {
			def = def[idx+1 : end]
		}
	}
	var cols []pgd.ColRef
	for _, part := range strings.Split(def, ",") {
		name := strings.TrimSpace(part)
		name = strings.Trim(name, `"`)
		if name != "" {
			cols = append(cols, pgd.ColRef{Name: name})
		}
	}
	return cols
}

func (i *introspector) introspectMatViewIndexes(project *pgd.Project) {
	type mvIdx struct {
		SchemaName string
		TableName  string
		IndexName  string
		IsUnique   bool
		Method     string
		Def        string
		Predicate  string
	}
	var idxs []mvIdx
	_, err := i.db.Query(&idxs, fmt.Sprintf(`
		SELECT n.nspname AS schema_name, c.relname AS table_name,
		       ic.relname AS index_name, ix.indisunique AS is_unique,
		       am.amname AS method,
		       pg_get_indexdef(ix.indexrelid) AS def,
		       COALESCE(pg_get_expr(ix.indpred, ix.indrelid), '') AS predicate
		FROM pg_index ix
		JOIN pg_class ic ON ic.oid = ix.indexrelid
		JOIN pg_class c ON c.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_am am ON am.oid = ic.relam
		WHERE c.relkind = 'm'
		  AND NOT ix.indisprimary
		  %s
		ORDER BY n.nspname, ic.relname
	`, i.schemaFilter()))
	if err != nil || len(idxs) == 0 {
		return
	}

	// Find schema and add indexes
	schemaMap := make(map[string]*pgd.Schema)
	for si := range project.Schemas {
		schemaMap[project.Schemas[si].Name] = &project.Schemas[si]
	}
	for _, idx := range idxs {
		s, ok := schemaMap[idx.SchemaName]
		if !ok {
			continue
		}
		pgIdx := convertIndex(idx.TableName, pgIndex{
			IndexName: idx.IndexName,
			IsUnique:  idx.IsUnique,
			Method:    idx.Method,
			Def:       idx.Def,
			Predicate: idx.Predicate,
		})
		if pgIdx != nil {
			s.Indexes = append(s.Indexes, *pgIdx)
		}
	}
}

func (i *introspector) queryTriggers() ([]pgTrigger, error) {
	var triggers []pgTrigger
	_, err := i.db.Query(&triggers, fmt.Sprintf(`
		SELECT t.tgname AS name, c.relname AS table_name, n.nspname AS schema_name,
		       pg_get_triggerdef(t.oid) AS definition
		FROM pg_trigger t
		JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE NOT t.tgisinternal
		  %s
		ORDER BY n.nspname, c.relname, t.tgname
	`, i.schemaFilter()))
	return triggers, err
}

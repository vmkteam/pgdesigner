package pgre

import (
	"fmt"

	"github.com/go-pg/pg/v10"
)

// PreviewResult holds a lightweight catalog of database objects without full introspection.
type PreviewResult struct {
	Database          string
	PgVersion         string
	Schemas           []SchemaPreview
	Views             []ObjectPreview
	MatViews          []ObjectPreview
	Functions         []ObjectPreview
	Triggers          []ObjectPreview
	Enums             []ObjectPreview
	Domains           []ObjectPreview
	Sequences         []ObjectPreview
	Extensions        []ObjectPreview
	Roles             []RolePreview
	Grants            int
	DefaultPrivileges int
}

// SchemaPreview holds schema name and table summaries.
type SchemaPreview struct {
	Name   string
	Tables []TablePreview
}

// TablePreview holds lightweight table metadata.
type TablePreview struct {
	Name        string
	Columns     int
	Indexes     int
	FKs         int
	Partitioned bool
}

// ObjectPreview holds name and schema for a database object.
type ObjectPreview struct {
	Name   string
	Schema string
}

// RolePreview holds lightweight role metadata.
type RolePreview struct {
	Name    string
	Login   bool
	Members int
}

// Preview connects to a PostgreSQL database and returns a lightweight catalog of objects.
func Preview(dsn string) (*PreviewResult, error) {
	db, err := connectDB(dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	p := &previewer{db: db}
	return p.preview()
}

type previewer struct {
	db *pg.DB
}

func (p *previewer) preview() (*PreviewResult, error) {
	r := &PreviewResult{}

	// Database name and version in one query
	if _, err := p.db.QueryOne(pg.Scan(&r.Database, &r.PgVersion),
		"SELECT current_database(), split_part(version(), ' ', 2)"); err != nil {
		return nil, fmt.Errorf("querying database info: %w", err)
	}

	if err := p.queryTables(r); err != nil {
		return nil, err
	}
	if err := p.queryObjects(r); err != nil {
		return nil, err
	}
	if err := p.queryExtensions(r); err != nil {
		return nil, err
	}
	if err := p.queryRoles(r); err != nil {
		return nil, err
	}

	return r, nil
}

func (p *previewer) queryTables(r *PreviewResult) error {
	var rows []struct {
		SchemaName  string `pg:"schema_name"`
		TableName   string `pg:"table_name"`
		Partitioned bool   `pg:"partitioned"`
		ColCount    int    `pg:"col_count"`
		IdxCount    int    `pg:"idx_count"`
		FKCount     int    `pg:"fk_count"`
	}
	_, err := p.db.Query(&rows, fmt.Sprintf(`
		SELECT n.nspname AS schema_name, c.relname AS table_name,
			c.relkind = 'p' AS partitioned,
			(SELECT count(*) FROM pg_attribute a WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped) AS col_count,
			(SELECT count(*) FROM pg_index i WHERE i.indrelid = c.oid AND NOT i.indisprimary) AS idx_count,
			(SELECT count(*) FROM pg_constraint con WHERE con.conrelid = c.oid AND con.contype = 'f') AS fk_count
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'p')
			%s
		ORDER BY n.nspname, c.relname
	`, defaultSchemaFilter))
	if err != nil {
		return fmt.Errorf("querying tables: %w", err)
	}

	schemaMap := make(map[string]*SchemaPreview)
	var schemaOrder []string
	for _, row := range rows {
		sp, ok := schemaMap[row.SchemaName]
		if !ok {
			sp = &SchemaPreview{Name: row.SchemaName}
			schemaMap[row.SchemaName] = sp
			schemaOrder = append(schemaOrder, row.SchemaName)
		}
		sp.Tables = append(sp.Tables, TablePreview{
			Name:        row.TableName,
			Columns:     row.ColCount,
			Indexes:     row.IdxCount,
			FKs:         row.FKCount,
			Partitioned: row.Partitioned,
		})
	}

	r.Schemas = make([]SchemaPreview, 0, len(schemaOrder))
	for _, name := range schemaOrder {
		r.Schemas = append(r.Schemas, *schemaMap[name])
	}
	return nil
}

func (p *previewer) queryObjects(r *PreviewResult) error {
	var rows []struct {
		Kind       string `pg:"kind"`
		SchemaName string `pg:"schema_name"`
		ObjName    string `pg:"obj_name"`
	}
	_, err := p.db.Query(&rows, fmt.Sprintf(`
		SELECT 'view' AS kind, n.nspname AS schema_name, c.relname AS obj_name
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'v' %[1]s
		UNION ALL
		SELECT 'matview', n.nspname, c.relname
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'm' %[1]s
		UNION ALL
		SELECT 'function', n.nspname, p.proname
		FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_language l ON l.oid = p.prolang
		WHERE p.prokind IN ('f','p') AND l.lanname != 'internal' %[1]s
		UNION ALL
		SELECT 'trigger', n.nspname, t.tgname
		FROM pg_trigger t JOIN pg_class c ON c.oid = t.tgrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE NOT t.tgisinternal %[1]s
		UNION ALL
		SELECT 'enum', n.nspname, t.typname
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typtype = 'e' %[1]s
		UNION ALL
		SELECT 'domain', n.nspname, t.typname
		FROM pg_type t JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typtype = 'd' %[1]s
		UNION ALL
		SELECT 'sequence', n.nspname, c.relname
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'S' %[1]s
		ORDER BY kind, schema_name, obj_name
	`, defaultSchemaFilter))
	if err != nil {
		return fmt.Errorf("querying objects: %w", err)
	}

	for _, row := range rows {
		obj := ObjectPreview{Name: row.ObjName, Schema: row.SchemaName}
		switch row.Kind {
		case "view":
			r.Views = append(r.Views, obj)
		case "matview":
			r.MatViews = append(r.MatViews, obj)
		case "function":
			r.Functions = append(r.Functions, obj)
		case "trigger":
			r.Triggers = append(r.Triggers, obj)
		case "enum":
			r.Enums = append(r.Enums, obj)
		case "domain":
			r.Domains = append(r.Domains, obj)
		case "sequence":
			r.Sequences = append(r.Sequences, obj)
		}
	}
	return nil
}

func (p *previewer) queryExtensions(r *PreviewResult) error {
	var rows []struct {
		Name   string `pg:"name"`
		Schema string `pg:"schema"`
	}
	_, err := p.db.Query(&rows, `
		SELECT e.extname AS name, n.nspname AS schema
		FROM pg_extension e
		JOIN pg_namespace n ON n.oid = e.extnamespace
		WHERE e.extname != 'plpgsql'
		ORDER BY e.extname
	`)
	if err != nil {
		return fmt.Errorf("querying extensions: %w", err)
	}
	for _, row := range rows {
		r.Extensions = append(r.Extensions, ObjectPreview{Name: row.Name, Schema: row.Schema})
	}
	return nil
}

func (p *previewer) queryRoles(r *PreviewResult) error {
	var rows []struct {
		Name    string `pg:"name"`
		Login   bool   `pg:"login"`
		Members int    `pg:"members"`
	}
	_, err := p.db.Query(&rows, `
		SELECT r.rolname AS name, r.rolcanlogin AS login,
			(SELECT count(*) FROM pg_auth_members m WHERE m.roleid = r.oid) AS members
		FROM pg_roles r
		WHERE r.rolname NOT LIKE 'pg_%'
			AND r.rolname != 'postgres'
			AND NOT r.rolsuper
		ORDER BY r.rolname
	`)
	if err != nil {
		return fmt.Errorf("querying roles: %w", err)
	}
	for _, row := range rows {
		r.Roles = append(r.Roles, RolePreview{Name: row.Name, Login: row.Login, Members: row.Members})
	}
	return nil
}

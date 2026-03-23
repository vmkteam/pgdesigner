package merge

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/vmkteam/pgdesigner/pkg/pgd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func proj(schemas ...pgd.Schema) *pgd.Project {
	return &pgd.Project{Schemas: schemas}
}

func projWithMeta(name string, schemas ...pgd.Schema) *pgd.Project {
	return &pgd.Project{
		ProjectMeta: pgd.ProjectMeta{Name: name},
		Schemas:     schemas,
	}
}

func sch(name string, tables ...pgd.Table) pgd.Schema {
	return pgd.Schema{Name: name, Tables: tables}
}

func schWithIdx(name string, tables []pgd.Table, indexes []pgd.Index) pgd.Schema {
	return pgd.Schema{Name: name, Tables: tables, Indexes: indexes}
}

func tbl(name string, cols ...pgd.Column) pgd.Table {
	return pgd.Table{Name: name, Columns: cols}
}

func tblWithPK(name string, pkName string, pkCols []string, cols ...pgd.Column) pgd.Table {
	return pgd.Table{Name: name, Columns: cols, PK: pk(pkName, pkCols...)}
}

func col(name, typ string) pgd.Column {
	return pgd.Column{Name: name, Type: typ}
}

func colNN(name, typ string) pgd.Column {
	return pgd.Column{Name: name, Type: typ, Nullable: "false"}
}

func pk(name string, cols ...string) *pgd.PrimaryKey {
	p := &pgd.PrimaryKey{Name: name}
	for _, c := range cols {
		p.Columns = append(p.Columns, pgd.ColRef{Name: c})
	}
	return p
}

func fk(name, toTable string, pairs ...string) pgd.ForeignKey {
	f := pgd.ForeignKey{Name: name, ToTable: toTable, OnDelete: "no action", OnUpdate: "no action"}
	for i := 0; i+1 < len(pairs); i += 2 {
		f.Columns = append(f.Columns, pgd.FKCol{Name: pairs[i], References: pairs[i+1]})
	}
	return f
}

func idx(name, table string, cols ...string) pgd.Index {
	i := pgd.Index{Name: name, Table: table}
	for _, c := range cols {
		i.Columns = append(i.Columns, pgd.ColRef{Name: c})
	}
	return i
}

func layout(entities ...pgd.LayoutEntity) pgd.Layouts {
	return pgd.Layouts{Layouts: []pgd.Layout{{
		Name: "Default Diagram", Default: "true", Entities: entities,
	}}}
}

func entity(schema, table string, x, y int) pgd.LayoutEntity {
	return pgd.LayoutEntity{Schema: schema, Table: table, X: x, Y: y}
}

// --- unit tests ---

func TestMerge_DisjointSchemas(t *testing.T) {
	base := proj(sch("public", tbl("users", col("id", "integer"))))
	overlay := proj(sch("app", tbl("orders", col("id", "integer"))))

	got, stats := Merge(base, overlay, Options{})

	require.Len(t, got.Schemas, 2)
	assert.Equal(t, "app", got.Schemas[0].Name)    // overlay first
	assert.Equal(t, "public", got.Schemas[1].Name) // then base
	assert.Equal(t, 0, stats.Common)
	assert.Equal(t, 1, stats.OnlyBase)
	assert.Equal(t, 1, stats.OnlyOverlay)
	assert.Equal(t, 2, stats.Total)
}

func TestMerge_SameSchema_DisjointTables(t *testing.T) {
	base := proj(sch("public", tbl("users", col("id", "integer"))))
	overlay := proj(sch("public", tbl("orders", col("id", "integer"))))

	got, stats := Merge(base, overlay, Options{})

	require.Len(t, got.Schemas, 1)
	require.Len(t, got.Schemas[0].Tables, 2)
	assert.Equal(t, "orders", got.Schemas[0].Tables[0].Name) // overlay first
	assert.Equal(t, "users", got.Schemas[0].Tables[1].Name)  // base-only
	assert.Equal(t, 0, stats.Common)
	assert.Equal(t, 1, stats.OnlyBase)
	assert.Equal(t, 1, stats.OnlyOverlay)
}

func TestMerge_SameTable_OverlayWins(t *testing.T) {
	base := proj(sch("public", tbl("users", col("id", "integer"), col("name", "text"))))
	overlay := proj(sch("public", tbl("users", col("id", "bigint"), col("email", "text"))))

	got, stats := Merge(base, overlay, Options{})

	require.Len(t, got.Schemas[0].Tables, 1)
	tUsers := got.Schemas[0].Tables[0]

	// overlay columns first, then base-only appended
	require.Len(t, tUsers.Columns, 3)
	assert.Equal(t, "id", tUsers.Columns[0].Name)
	assert.Equal(t, "bigint", tUsers.Columns[0].Type) // overlay wins
	assert.Equal(t, "email", tUsers.Columns[1].Name)
	assert.Equal(t, "name", tUsers.Columns[2].Name) // base-only appended

	assert.Equal(t, 1, stats.Common)
	assert.Equal(t, 0, stats.OnlyBase)
	assert.Equal(t, 0, stats.OnlyOverlay)
}

func TestMerge_PKOverlayWins(t *testing.T) {
	base := proj(sch("public", tblWithPK("users", "pk_users", []string{"id"}, colNN("id", "integer"))))
	overlay := proj(sch("public", tbl("users", colNN("id", "integer"))))

	got, _ := Merge(base, overlay, Options{})

	// overlay has no PK, so base PK is used
	tUsers := got.Schemas[0].Tables[0]
	require.NotNil(t, tUsers.PK)
	assert.Equal(t, "pk_users", tUsers.PK.Name)
}

func TestMerge_PKOverlaySet(t *testing.T) {
	base := proj(sch("public", tblWithPK("users", "pk_old", []string{"id"}, colNN("id", "integer"))))
	overlay := proj(sch("public", tblWithPK("users", "pk_new", []string{"id"}, colNN("id", "integer"))))

	got, _ := Merge(base, overlay, Options{})

	// overlay PK wins
	tUsers := got.Schemas[0].Tables[0]
	require.NotNil(t, tUsers.PK)
	assert.Equal(t, "pk_new", tUsers.PK.Name)
}

func TestMerge_FKUnion(t *testing.T) {
	base := proj(sch("public",
		tbl("orders", col("id", "integer"), col("user_id", "integer")),
	))
	base.Schemas[0].Tables[0].FKs = []pgd.ForeignKey{fk("fk_user", "users", "user_id", "id")}

	overlay := proj(sch("public",
		tbl("orders", col("id", "integer"), col("product_id", "integer")),
	))
	overlay.Schemas[0].Tables[0].FKs = []pgd.ForeignKey{fk("fk_product", "products", "product_id", "id")}

	got, _ := Merge(base, overlay, Options{})

	tOrders := got.Schemas[0].Tables[0]
	require.Len(t, tOrders.FKs, 2)
	assert.Equal(t, "fk_product", tOrders.FKs[0].Name) // overlay first
	assert.Equal(t, "fk_user", tOrders.FKs[1].Name)    // base-only
}

func TestMerge_IndexesUnion(t *testing.T) {
	base := proj(schWithIdx("public",
		[]pgd.Table{tbl("users", col("id", "integer"))},
		[]pgd.Index{idx("idx_users_id", "users", "id")},
	))
	overlay := proj(schWithIdx("public",
		[]pgd.Table{tbl("users", col("id", "integer"))},
		[]pgd.Index{idx("idx_users_name", "users", "name")},
	))

	got, _ := Merge(base, overlay, Options{})

	require.Len(t, got.Schemas[0].Indexes, 2)
	assert.Equal(t, "idx_users_name", got.Schemas[0].Indexes[0].Name)
	assert.Equal(t, "idx_users_id", got.Schemas[0].Indexes[1].Name)
}

func TestMerge_Extensions(t *testing.T) {
	base := &pgd.Project{Extensions: []pgd.Extension{{Name: "pgcrypto"}}}
	overlay := &pgd.Project{Extensions: []pgd.Extension{{Name: "uuid-ossp"}, {Name: "pgcrypto"}}}

	got, _ := Merge(base, overlay, Options{})

	require.Len(t, got.Extensions, 2)
	assert.Equal(t, "uuid-ossp", got.Extensions[0].Name)
	assert.Equal(t, "pgcrypto", got.Extensions[1].Name) // overlay wins, no duplicate
}

func TestMerge_Sequences(t *testing.T) {
	base := &pgd.Project{Sequences: []pgd.Sequence{{Name: "seq_a"}}}
	overlay := &pgd.Project{Sequences: []pgd.Sequence{{Name: "seq_b"}}}

	got, _ := Merge(base, overlay, Options{})

	require.Len(t, got.Sequences, 2)
	assert.Equal(t, "seq_b", got.Sequences[0].Name)
	assert.Equal(t, "seq_a", got.Sequences[1].Name)
}

func TestMerge_Types(t *testing.T) {
	base := &pgd.Project{Types: &pgd.Types{
		Enums: []pgd.Enum{{Name: "status", Labels: []string{"active"}}},
	}}
	overlay := &pgd.Project{Types: &pgd.Types{
		Enums:   []pgd.Enum{{Name: "role", Labels: []string{"admin"}}},
		Domains: []pgd.Domain{{Name: "email", Type: "text"}},
	}}

	got, _ := Merge(base, overlay, Options{})

	require.NotNil(t, got.Types)
	require.Len(t, got.Types.Enums, 2)
	assert.Equal(t, "role", got.Types.Enums[0].Name)
	assert.Equal(t, "status", got.Types.Enums[1].Name)
	require.Len(t, got.Types.Domains, 1)
}

func TestMerge_TypesBothNil(t *testing.T) {
	base := &pgd.Project{}
	overlay := &pgd.Project{}

	got, _ := Merge(base, overlay, Options{})
	assert.Nil(t, got.Types)
}

func TestMerge_Views(t *testing.T) {
	base := &pgd.Project{Views: &pgd.Views{
		Views: []pgd.View{{Name: "v_users", Query: "SELECT 1"}},
	}}
	overlay := &pgd.Project{Views: &pgd.Views{
		Views: []pgd.View{{Name: "v_orders", Query: "SELECT 2"}},
	}}

	got, _ := Merge(base, overlay, Options{})

	require.NotNil(t, got.Views)
	require.Len(t, got.Views.Views, 2)
	assert.Equal(t, "v_orders", got.Views.Views[0].Name)
	assert.Equal(t, "v_users", got.Views.Views[1].Name)
}

func TestMerge_ViewsBothNil(t *testing.T) {
	got, _ := Merge(&pgd.Project{}, &pgd.Project{}, Options{})
	assert.Nil(t, got.Views)
}

func TestMerge_Comments(t *testing.T) {
	base := &pgd.Project{Comments: []pgd.Comment{{On: "table", Table: "users", Value: "old"}}}
	overlay := &pgd.Project{Comments: []pgd.Comment{{On: "table", Table: "orders", Value: "new"}}}

	got, _ := Merge(base, overlay, Options{})
	require.Len(t, got.Comments, 2)
}

func TestMerge_Partitioning(t *testing.T) {
	base := proj(sch("public", pgd.Table{
		Name:    "events",
		Columns: []pgd.Column{col("id", "integer"), col("created_at", "timestamptz")},
		PartitionBy: &pgd.PartitionBy{
			Type:    "range",
			Columns: []pgd.ColRef{{Name: "created_at"}},
		},
		Partitions: []pgd.Partition{{Name: "events_2024", Bound: "FOR VALUES FROM ('2024-01-01') TO ('2025-01-01')"}},
	}))

	overlay := proj(sch("public", pgd.Table{
		Name:    "events",
		Columns: []pgd.Column{col("id", "integer"), col("created_at", "timestamptz")},
		PartitionBy: &pgd.PartitionBy{
			Type:    "range",
			Columns: []pgd.ColRef{{Name: "created_at"}},
		},
		Partitions: []pgd.Partition{{Name: "events_2025", Bound: "FOR VALUES FROM ('2025-01-01') TO ('2026-01-01')"}},
	}))

	got, _ := Merge(base, overlay, Options{})

	tEvents := got.Schemas[0].Tables[0]
	require.NotNil(t, tEvents.PartitionBy)
	assert.Equal(t, "range", tEvents.PartitionBy.Type)
	require.Len(t, tEvents.Partitions, 2)
	assert.Equal(t, "events_2025", tEvents.Partitions[0].Name) // overlay first
	assert.Equal(t, "events_2024", tEvents.Partitions[1].Name)
}

func TestMerge_PartitioningOneSided(t *testing.T) {
	base := proj(sch("public", tbl("events", col("id", "integer"))))
	overlay := proj(sch("public", pgd.Table{
		Name:    "events",
		Columns: []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{
			Type:    "list",
			Columns: []pgd.ColRef{{Name: "region"}},
		},
	}))

	got, _ := Merge(base, overlay, Options{})

	tEvents := got.Schemas[0].Tables[0]
	require.NotNil(t, tEvents.PartitionBy)
	assert.Equal(t, "list", tEvents.PartitionBy.Type)
}

func TestMerge_TableComment(t *testing.T) {
	base := proj(sch("public", pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{col("id", "integer")},
		Comment: "base comment",
	}))
	overlay := proj(sch("public", tbl("users", col("id", "integer"))))

	got, _ := Merge(base, overlay, Options{})

	// overlay comment is empty, base comment preserved
	assert.Equal(t, "base comment", got.Schemas[0].Tables[0].Comment)
}

func TestMerge_TableCommentOverlayWins(t *testing.T) {
	base := proj(sch("public", pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{col("id", "integer")},
		Comment: "base comment",
	}))
	overlay := proj(sch("public", pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{col("id", "integer")},
		Comment: "overlay comment",
	}))

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, "overlay comment", got.Schemas[0].Tables[0].Comment)
}

func TestMerge_Layouts_Both(t *testing.T) {
	base := projWithMeta("base", sch("public", tbl("users", col("id", "integer"))))
	base.Layouts = layout(entity("public", "users", 10, 20))

	overlay := projWithMeta("overlay", sch("public", tbl("orders", col("id", "integer"))))
	overlay.Layouts = layout(entity("public", "orders", 100, 200))

	got, _ := Merge(base, overlay, Options{Layout: "both"})

	require.Len(t, got.Layouts.Layouts, 1)
	entities := got.Layouts.Layouts[0].Entities
	require.Len(t, entities, 2)

	pos := map[string]pgd.LayoutEntity{}
	for _, e := range entities {
		pos[e.Table] = e
	}
	assert.Equal(t, 10, pos["users"].X)
	assert.Equal(t, 100, pos["orders"].X)
}

func TestMerge_Layouts_OverlayWinsConflict(t *testing.T) {
	base := projWithMeta("base", sch("public", tbl("users", col("id", "integer"))))
	base.Layouts = layout(entity("public", "users", 10, 20))

	overlay := projWithMeta("overlay", sch("public", tbl("users", col("id", "integer"))))
	overlay.Layouts = layout(entity("public", "users", 99, 88))

	got, _ := Merge(base, overlay, Options{Layout: "both"})

	entities := got.Layouts.Layouts[0].Entities
	require.Len(t, entities, 1)
	assert.Equal(t, 99, entities[0].X) // overlay wins
	assert.Equal(t, 88, entities[0].Y)
}

func TestMerge_Layouts_BaseMode(t *testing.T) {
	base := projWithMeta("base")
	base.Layouts = layout(entity("public", "users", 10, 20))

	overlay := projWithMeta("overlay")
	overlay.Layouts = layout(entity("public", "users", 99, 88))

	got, _ := Merge(base, overlay, Options{Layout: "base"})

	entities := got.Layouts.Layouts[0].Entities
	require.Len(t, entities, 1)
	assert.Equal(t, 10, entities[0].X) // base wins
}

func TestMerge_Layouts_OverlayMode(t *testing.T) {
	base := projWithMeta("base")
	base.Layouts = layout(entity("public", "users", 10, 20))

	overlay := projWithMeta("overlay")
	overlay.Layouts = layout(entity("public", "users", 99, 88))

	got, _ := Merge(base, overlay, Options{Layout: "overlay"})

	entities := got.Layouts.Layouts[0].Entities
	require.Len(t, entities, 1)
	assert.Equal(t, 99, entities[0].X) // overlay wins
}

func TestMerge_OptionsName(t *testing.T) {
	base := projWithMeta("base")
	overlay := projWithMeta("overlay")

	got, _ := Merge(base, overlay, Options{Name: "merged"})
	assert.Equal(t, "merged", got.ProjectMeta.Name)
}

func TestMerge_OptionsNameDefault(t *testing.T) {
	base := projWithMeta("base")
	overlay := projWithMeta("overlay")

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, "overlay", got.ProjectMeta.Name) // overlay wins
}

func TestMerge_DefaultSchema(t *testing.T) {
	base := &pgd.Project{DefaultSchema: "public"}
	overlay := &pgd.Project{DefaultSchema: "app"}

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, "app", got.DefaultSchema)
}

func TestMerge_DefaultSchemaFallback(t *testing.T) {
	base := &pgd.Project{DefaultSchema: "public"}
	overlay := &pgd.Project{}

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, "public", got.DefaultSchema) // falls back to base
}

func TestMerge_PgVersion(t *testing.T) {
	base := &pgd.Project{PgVersion: "16"}
	overlay := &pgd.Project{PgVersion: "18"}

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, "18", got.PgVersion)
}

func TestMerge_Version(t *testing.T) {
	base := &pgd.Project{Version: 1}
	overlay := &pgd.Project{Version: 2}

	got, _ := Merge(base, overlay, Options{})
	assert.Equal(t, 2, got.Version)
}

func TestMerge_UniqueConstraints(t *testing.T) {
	base := proj(sch("public", pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{col("id", "integer"), col("email", "text")},
		Uniques: []pgd.Unique{{Name: "uq_email", Columns: []pgd.ColRef{{Name: "email"}}}},
	}))
	overlay := proj(sch("public", pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{col("id", "integer"), col("login", "text")},
		Uniques: []pgd.Unique{{Name: "uq_login", Columns: []pgd.ColRef{{Name: "login"}}}},
	}))

	got, _ := Merge(base, overlay, Options{})

	tUsers := got.Schemas[0].Tables[0]
	require.Len(t, tUsers.Uniques, 2)
	assert.Equal(t, "uq_login", tUsers.Uniques[0].Name)
	assert.Equal(t, "uq_email", tUsers.Uniques[1].Name)
}

func TestMerge_CheckConstraints(t *testing.T) {
	base := proj(sch("public", pgd.Table{
		Name:   "users",
		Checks: []pgd.Check{{Name: "chk_age", Expression: "age > 0"}},
	}))
	overlay := proj(sch("public", pgd.Table{
		Name:   "users",
		Checks: []pgd.Check{{Name: "chk_name", Expression: "name <> ''"}},
	}))

	got, _ := Merge(base, overlay, Options{})

	tUsers := got.Schemas[0].Tables[0]
	require.Len(t, tUsers.Checks, 2)
	assert.Equal(t, "chk_name", tUsers.Checks[0].Name)
	assert.Equal(t, "chk_age", tUsers.Checks[1].Name)
}

func TestMerge_Functions(t *testing.T) {
	base := &pgd.Project{Functions: []pgd.Function{{Name: "func_a", Language: "sql"}}}
	overlay := &pgd.Project{Functions: []pgd.Function{{Name: "func_b", Language: "plpgsql"}}}

	got, _ := Merge(base, overlay, Options{})

	require.Len(t, got.Functions, 2)
	assert.Equal(t, "func_b", got.Functions[0].Name)
	assert.Equal(t, "func_a", got.Functions[1].Name)
}

func TestMerge_Triggers(t *testing.T) {
	base := &pgd.Project{Triggers: []pgd.Trigger{{Name: "trig_a"}}}
	overlay := &pgd.Project{Triggers: []pgd.Trigger{{Name: "trig_b"}}}

	got, _ := Merge(base, overlay, Options{})

	require.Len(t, got.Triggers, 2)
	assert.Equal(t, "trig_b", got.Triggers[0].Name)
	assert.Equal(t, "trig_a", got.Triggers[1].Name)
}

// --- integration tests with real pgd files ---

func loadProject(t *testing.T, path string) *pgd.Project {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var p pgd.Project
	require.NoError(t, xml.Unmarshal(data, &p))
	return &p
}

func TestMerge_RealFiles_ChinookNorthwind(t *testing.T) {
	chinook := loadProject(t, "../../format/sql/testdata/chinook.pgd")
	northwind := loadProject(t, "../../format/sql/testdata/northwind.pgd")

	got, stats := Merge(chinook, northwind, Options{Name: "merged"})

	assert.Equal(t, "merged", got.ProjectMeta.Name)
	assert.Positive(t, stats.Total)
	// all tables should be present (disjoint datasets = sum of both)
	assert.Equal(t, stats.OnlyBase+stats.OnlyOverlay+stats.Common, stats.Total)

	// count total tables
	var totalTables int
	for _, s := range got.Schemas {
		totalTables += len(s.Tables)
	}
	assert.Equal(t, stats.Total, totalTables)
}

func TestMerge_RealFiles_SelfMerge(t *testing.T) {
	chinook := loadProject(t, "../../format/sql/testdata/chinook.pgd")

	got, stats := Merge(chinook, chinook, Options{})

	// self-merge: all tables are Common, none OnlyBase/OnlyOverlay
	assert.Equal(t, 0, stats.OnlyBase)
	assert.Equal(t, 0, stats.OnlyOverlay)
	assert.Equal(t, stats.Total, stats.Common)

	// same number of tables
	var baseTables, mergedTables int
	for _, s := range chinook.Schemas {
		baseTables += len(s.Tables)
	}
	for _, s := range got.Schemas {
		mergedTables += len(s.Tables)
	}
	assert.Equal(t, baseTables, mergedTables)
}

func TestMerge_RealFiles_Adventureworks(t *testing.T) {
	aw := loadProject(t, "../../format/sql/testdata/adventureworks.pgd")
	chinook := loadProject(t, "../../format/sql/testdata/chinook.pgd")

	got, stats := Merge(aw, chinook, Options{})

	// adventureworks has multiple schemas, chinook has public
	assert.GreaterOrEqual(t, len(got.Schemas), 2)
	assert.Positive(t, stats.Total)
	assert.Equal(t, stats.OnlyBase+stats.OnlyOverlay+stats.Common, stats.Total)
}

package diff

import (
	"encoding/xml"
	"os"
	"slices"
	"strings"
	"testing"

	pgquery "github.com/wasilibs/go-pgquery"

	"github.com/vmkteam/pgdesigner/pkg/pgd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func project(schemas ...pgd.Schema) *pgd.Project {
	return &pgd.Project{Schemas: schemas}
}

func schema(name string, tables []pgd.Table, indexes []pgd.Index) pgd.Schema {
	return pgd.Schema{Name: name, Tables: tables, Indexes: indexes}
}

func col(name, typ string) pgd.Column {
	return pgd.Column{Name: name, Type: typ}
}

func colNN(name, typ string) pgd.Column {
	return pgd.Column{Name: name, Type: typ, Nullable: "false"}
}

func colDefault(name, typ, def string) pgd.Column {
	return pgd.Column{Name: name, Type: typ, Default: def}
}

func colIdentity(name, typ, gen string) pgd.Column {
	return pgd.Column{Name: name, Type: typ, Nullable: "false", Identity: &pgd.Identity{Generated: gen}}
}

func pk(name string, cols ...string) *pgd.PrimaryKey {
	p := &pgd.PrimaryKey{Name: name}
	for _, c := range cols {
		p.Columns = append(p.Columns, pgd.ColRef{Name: c})
	}
	return p
}

func fk(name, toTable, onDel string, pairs ...string) pgd.ForeignKey {
	f := pgd.ForeignKey{Name: name, ToTable: toTable, OnDelete: onDel, OnUpdate: "no action"}
	for i := 0; i+1 < len(pairs); i += 2 {
		f.Columns = append(f.Columns, pgd.FKCol{Name: pairs[i], References: pairs[i+1]})
	}
	return f
}

// changeSQLs returns the SQL of each change.
func changeSQLs(r *DiffResult) []string {
	var out []string
	for _, c := range r.Changes {
		out = append(out, c.SQL)
	}
	return out
}

func idx(name, table string, cols ...string) pgd.Index {
	i := pgd.Index{Name: name, Table: table}
	for _, c := range cols {
		i.Keys = append(i.Keys, pgd.IndexKey{Name: c})
	}
	return i
}

// --- Tests ---

func TestDiff_Empty(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	r := Diff(old, old)
	assert.Empty(t, r.Changes)
}

func TestDiff_AddTable(t *testing.T) {
	old := project(schema("public", nil, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:    "users",
		Columns: []pgd.Column{colNN("id", "integer"), col("name", "text")},
		PK:      pk("pk_users", "id"),
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "table", r.Changes[0].Object)
	assert.Equal(t, "add", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `CREATE TABLE "users"`)
	assert.Contains(t, r.Changes[0].SQL, `"id" integer NOT NULL`)
}

func TestDiff_DropTable(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	updated := project(schema("public", nil, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "drop", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `DROP TABLE "users"`)
	assert.True(t, r.HasHazards())
	assert.Equal(t, "DELETES_DATA", r.Changes[0].Hazards[0].Code)
}

func TestDiff_AddColumn(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer"), col("email", "text")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	c := r.Changes[0]
	assert.Equal(t, "column", c.Object)
	assert.Equal(t, "add", c.Action)
	assert.Equal(t, "email", c.Name)
	assert.Contains(t, c.SQL, `ADD COLUMN "email" text`)
}

func TestDiff_AddColumnNotNullNoDefault(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer"), colNN("x", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "BACKFILL_REQUIRED", r.Changes[0].Hazards[0].Code)
}

func TestDiff_DropColumn(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer"), col("email", "text")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "users", Columns: []pgd.Column{col("id", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	c := r.Changes[0]
	assert.Equal(t, "column", c.Object)
	assert.Equal(t, "drop", c.Action)
	assert.Contains(t, c.SQL, `DROP COLUMN "email"`)
	assert.Equal(t, "DELETES_DATA", c.Hazards[0].Code)
}

func TestDiff_AlterColumnType(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{{Name: "x", Type: "varchar", Length: 100}}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{{Name: "x", Type: "varchar", Length: 255}}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `TYPE varchar(255)`)
	assert.Empty(t, r.Changes[0].Hazards) // varchar → varchar is compatible
}

func TestDiff_AlterColumnTypeIncompatible(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "text")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "TABLE_REWRITE", r.Changes[0].Hazards[0].Code)
}

func TestDiff_AlterColumnNullable(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colNN("x", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `SET NOT NULL`)
}

func TestDiff_AlterColumnDropNotNull(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colNN("x", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP NOT NULL`)
}

func TestDiff_AlterColumnDefault(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colDefault("x", "integer", "0")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `SET DEFAULT 0`)
}

func TestDiff_AlterColumnDropDefault(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colDefault("x", "integer", "0")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP DEFAULT`)
}

func TestDiff_AddIdentity(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colNN("id", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colIdentity("id", "integer", "always")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `ADD GENERATED ALWAYS AS IDENTITY`)
}

func TestDiff_DropIdentity(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colIdentity("id", "integer", "always")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colNN("id", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP IDENTITY`)
}

func TestDiff_ChangeIdentity(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colIdentity("id", "integer", "by-default")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colIdentity("id", "integer", "always")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `SET GENERATED ALWAYS`)
}

func TestDiff_AddIndex(t *testing.T) {
	old := project(schema("public", nil, nil))
	updated := project(schema("public", nil, []pgd.Index{idx("idx_email", "users", "email")}))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "index", r.Changes[0].Object)
	assert.Equal(t, "add", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `CREATE INDEX "idx_email"`)
}

func TestDiff_DropIndex(t *testing.T) {
	old := project(schema("public", nil, []pgd.Index{idx("idx_email", "users", "email")}))
	updated := project(schema("public", nil, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP INDEX "idx_email"`)
}

func TestDiff_ModifyIndex(t *testing.T) {
	old := project(schema("public", nil, []pgd.Index{idx("idx_x", "t", "a", "b")}))
	updated := project(schema("public", nil, []pgd.Index{idx("idx_x", "t", "a")}))

	r := Diff(old, updated)
	// should be DROP + CREATE
	require.Len(t, r.Changes, 2)
	assert.Equal(t, "drop", r.Changes[0].Action)
	assert.Equal(t, "add", r.Changes[1].Action)
}

func TestDiff_AddFK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "orders", Columns: []pgd.Column{col("id", "integer"), col("userId", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:    "orders",
		Columns: []pgd.Column{col("id", "integer"), col("userId", "integer")},
		FKs:     []pgd.ForeignKey{fk("fk_orders_users", "users", "restrict", "userId", "userId")},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "fk", r.Changes[0].Object)
	assert.Equal(t, "add", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `FOREIGN KEY`)
}

func TestDiff_DropFK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:    "orders",
		Columns: []pgd.Column{col("id", "integer"), col("userId", "integer")},
		FKs:     []pgd.ForeignKey{fk("fk_orders_users", "users", "restrict", "userId", "userId")},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "orders", Columns: []pgd.Column{col("id", "integer"), col("userId", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP CONSTRAINT "fk_orders_users"`)
}

func TestDiff_ModifyFK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:    "t",
		Columns: []pgd.Column{col("id", "integer")},
		FKs:     []pgd.ForeignKey{fk("fk1", "a", "restrict", "id", "id")},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:    "t",
		Columns: []pgd.Column{col("id", "integer")},
		FKs:     []pgd.ForeignKey{fk("fk1", "a", "cascade", "id", "id")},
	}}, nil))

	r := Diff(old, updated)
	// DROP + ADD
	require.Len(t, r.Changes, 2)
	assert.Equal(t, "drop", r.Changes[0].Action)
	assert.Equal(t, "add", r.Changes[1].Action)
}

func TestDiff_AddPK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer")}, PK: pk("pk_t", "id")}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `ADD CONSTRAINT "pk_t" PRIMARY KEY ("id")`)
}

func TestDiff_DropPK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer")}, PK: pk("pk_t", "id")}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP CONSTRAINT "pk_t"`)
}

func TestDiff_ModifyPK(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("a", "integer"), col("b", "integer")}, PK: pk("pk_t", "a")}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("a", "integer"), col("b", "integer")}, PK: pk("pk_t", "a", "b")}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 2) // DROP + ADD
	assert.Contains(t, r.Changes[0].SQL, `DROP CONSTRAINT`)
	assert.Contains(t, r.Changes[1].SQL, `PRIMARY KEY ("a", "b")`)
}

func TestDiff_AddUnique(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("email", "text")}}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:    "t",
		Columns: []pgd.Column{col("email", "text")},
		Uniques: []pgd.Unique{{Name: "uq_email", Columns: []pgd.ColRef{{Name: "email"}}}},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `ADD CONSTRAINT "uq_email" UNIQUE("email")`)
}

func TestDiff_AddCheck(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:    "t",
		Columns: []pgd.Column{col("x", "integer")},
		Checks:  []pgd.Check{{Name: "chk_positive", Expression: "x > 0"}},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `ADD CONSTRAINT "chk_positive" CHECK (x > 0)`)
}

func TestDiff_AddEnumValue(t *testing.T) {
	old := &pgd.Project{Types: &pgd.Types{Enums: []pgd.Enum{{Name: "status", Labels: []string{"active", "deleted"}}}}}
	updated := &pgd.Project{Types: &pgd.Types{Enums: []pgd.Enum{{Name: "status", Labels: []string{"active", "deleted", "archived"}}}}}

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `ADD VALUE 'archived'`)
}

func TestDiff_SQL(t *testing.T) {
	old := project(schema("public",
		[]pgd.Table{{Name: "users", Columns: []pgd.Column{colNN("id", "integer"), col("name", "text")}}},
		[]pgd.Index{idx("idx_name", "users", "name")},
	))
	updated := project(schema("public",
		[]pgd.Table{{
			Name:    "users",
			Columns: []pgd.Column{colNN("id", "integer"), col("name", "text"), colDefault("email", "text", "''")},
			FKs:     []pgd.ForeignKey{fk("fk_users_org", "orgs", "restrict", "id", "id")},
		}},
		[]pgd.Index{idx("idx_name", "users", "name"), idx("idx_email", "users", "email")},
	))

	r := Diff(old, updated)
	sql := r.SQL()

	assert.Contains(t, sql, `ADD COLUMN "email"`)
	assert.Contains(t, sql, `CREATE INDEX "idx_email"`)
	assert.Contains(t, sql, `FOREIGN KEY`)
	assert.NotContains(t, sql, `DROP`) // nothing dropped

	// verify order: columns before indexes before FK
	colPos := strings.Index(sql, "ADD COLUMN")
	idxPos := strings.Index(sql, "CREATE INDEX")
	fkPos := strings.Index(sql, "FOREIGN KEY")
	assert.Less(t, colPos, idxPos, "ADD COLUMN before CREATE INDEX")
	assert.Less(t, idxPos, fkPos, "CREATE INDEX before ADD FK")
}

func TestDiff_IndexWhereChanged(t *testing.T) {
	oldIdx := pgd.Index{Name: "idx_x", Table: "t", Unique: "true", Keys: []pgd.IndexKey{{Name: "workspaceId"}, {Name: "statusId"}}}
	newIdx := pgd.Index{Name: "idx_x", Table: "t", Unique: "true", Keys: []pgd.IndexKey{{Name: "workspaceId"}}, Where: &pgd.WhereClause{Value: `"statusId" IN (1, 2)`}}

	old := project(schema("public", nil, []pgd.Index{oldIdx}))
	updated := project(schema("public", nil, []pgd.Index{newIdx}))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 2) // DROP + CREATE
	assert.Equal(t, "drop", r.Changes[0].Action)
	assert.Equal(t, "add", r.Changes[1].Action)
	assert.Contains(t, r.Changes[1].SQL, `WHERE`)
}

func TestDiff_AddPartition(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer"), col("dt", "timestamptz")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "dt"}}},
		Partitions: []pgd.Partition{
			{Name: "payment_p1", Bound: "FOR VALUES FROM ('2024-01-01') TO ('2024-02-01')"},
		},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer"), col("dt", "timestamptz")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "dt"}}},
		Partitions: []pgd.Partition{
			{Name: "payment_p1", Bound: "FOR VALUES FROM ('2024-01-01') TO ('2024-02-01')"},
			{Name: "payment_p2", Bound: "FOR VALUES FROM ('2024-02-01') TO ('2024-03-01')"},
		},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "partition", r.Changes[0].Object)
	assert.Equal(t, "add", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `CREATE TABLE "payment_p2" PARTITION OF "payment"`)
	assert.Contains(t, r.Changes[0].SQL, "FOR VALUES FROM")
}

func TestDiff_DropPartition(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
		Partitions: []pgd.Partition{
			{Name: "payment_p1", Bound: "FOR VALUES FROM (1) TO (100)"},
			{Name: "payment_p2", Bound: "FOR VALUES FROM (100) TO (200)"},
		},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
		Partitions: []pgd.Partition{
			{Name: "payment_p1", Bound: "FOR VALUES FROM (1) TO (100)"},
		},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "partition", r.Changes[0].Object)
	assert.Equal(t, "drop", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, `DETACH PARTITION "payment_p2"`)
}

func TestDiff_ChangePartitionBound(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
		Partitions:  []pgd.Partition{{Name: "payment_p1", Bound: "FOR VALUES FROM (1) TO (100)"}},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
		Partitions:  []pgd.Partition{{Name: "payment_p1", Bound: "FOR VALUES FROM (1) TO (200)"}},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Equal(t, "partition", r.Changes[0].Object)
	assert.Equal(t, "alter", r.Changes[0].Action)
	assert.Contains(t, r.Changes[0].SQL, "DETACH")
	assert.Contains(t, r.Changes[0].SQL, "ATTACH")
}

func TestDiff_AddPartitionBy(t *testing.T) {
	old := project(schema("public", []pgd.Table{{
		Name:    "payment",
		Columns: []pgd.Column{col("id", "integer")},
	}}, nil))
	updated := project(schema("public", []pgd.Table{{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
	}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, "table recreation")
	assert.Equal(t, "TABLE_RECREATE", r.Changes[0].Hazards[0].Code)
}

func TestDiff_PartitionNoChange(t *testing.T) {
	tbl := pgd.Table{
		Name:        "payment",
		Columns:     []pgd.Column{col("id", "integer")},
		PartitionBy: &pgd.PartitionBy{Type: "range", Columns: []pgd.ColRef{{Name: "id"}}},
		Partitions:  []pgd.Partition{{Name: "payment_p1", Bound: "FOR VALUES FROM (1) TO (100)"}},
	}
	old := project(schema("public", []pgd.Table{tbl}, nil))
	updated := project(schema("public", []pgd.Table{tbl}, nil))

	r := Diff(old, updated)
	assert.Empty(t, r.Changes)
}

func TestDiff_Golden(t *testing.T) {
	tests := []struct {
		name   string
		dir    string
		checks []string // SQL fragments that must be in output
	}{
		{
			"add-column",
			"testdata/diff/add-column",
			[]string{`ADD COLUMN "channel" varchar(255) NOT NULL DEFAULT`},
		},
		{
			"move-column",
			"testdata/diff/move-column",
			[]string{`DROP COLUMN "rating"`, `ADD COLUMN "rating" numeric(3,1)`},
		},
		{
			"modify-index",
			"testdata/diff/modify-index",
			[]string{`DROP INDEX`, `CREATE UNIQUE INDEX`, `WHERE`},
		},
		{
			"add-table",
			"testdata/diff/add-table",
			[]string{`CREATE TABLE "notifications"`, `"notificationId" integer NOT NULL`, `PRIMARY KEY`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldData, err := os.ReadFile(tt.dir + "/old.pgd")
			require.NoError(t, err)
			newData, err := os.ReadFile(tt.dir + "/new.pgd")
			require.NoError(t, err)

			var oldP, newP pgd.Project
			require.NoError(t, xml.Unmarshal(oldData, &oldP))
			require.NoError(t, xml.Unmarshal(newData, &newP))

			r := Diff(&oldP, &newP)
			sql := r.SQL()

			// golden file comparison
			golden, err := os.ReadFile(tt.dir + "/generated.sql")
			require.NoError(t, err)
			assert.Equal(t, string(golden), sql, "golden mismatch")

			// semantic checks
			for _, check := range tt.checks {
				assert.Contains(t, sql, check, "missing: "+check)
			}
		})
	}
}

func TestDiff_NewTableFKsAndComments(t *testing.T) {
	people := pgd.Table{Name: "people", Columns: []pgd.Column{colNN("id", "integer")}, PK: pk("people_pkey", "id")}
	notes := pgd.Table{
		Name:    "notes",
		Comment: "notes of a person",
		Columns: []pgd.Column{colNN("id", "integer"), colNN("personId", "integer"), {Name: "body", Type: "text", Comment: "text", Storage: "external"}},
		PK:      pk("notes_pkey", "id"),
		FKs:     []pgd.ForeignKey{fk("fk_notes_personId", "people", "restrict", "personId", "id")},
	}
	old := project(schema("public", []pgd.Table{people}, nil))
	updated := project(schema("public", []pgd.Table{notes, people}, nil))

	sql := Diff(old, updated).SQL()
	assert.Contains(t, sql, `CREATE TABLE "notes"`)
	assert.Contains(t, sql, `ALTER TABLE "notes" ADD CONSTRAINT "fk_notes_personId" FOREIGN KEY ("personId")`)
	assert.Contains(t, sql, `COMMENT ON TABLE "notes" IS 'notes of a person';`)
	assert.Contains(t, sql, `COMMENT ON COLUMN "notes"."body" IS 'text';`)
	assert.Contains(t, sql, `ALTER TABLE "notes" ALTER COLUMN "body" SET STORAGE EXTERNAL;`)
	assert.Less(t, strings.Index(sql, `CREATE TABLE "notes"`), strings.Index(sql, `FOREIGN KEY`))
}

func TestDiff_NewColumnComment(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "people", Columns: []pgd.Column{col("id", "integer")}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "people", Columns: []pgd.Column{col("id", "integer"), {Name: "nick", Type: "text", Comment: "прозвище"}}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 2)
	assert.Equal(t, `ALTER TABLE "people" ADD COLUMN "nick" text;`, r.Changes[0].SQL)
	assert.Equal(t, `COMMENT ON COLUMN "people"."nick" IS 'прозвище';`, r.Changes[1].SQL)
}

func TestDiff_CommentsSection(t *testing.T) {
	tbl := pgd.Table{Name: "notes", Columns: []pgd.Column{col("id", "integer"), col("body", "text")}}
	old := project(schema("public", []pgd.Table{tbl}, nil))
	old.Comments = []pgd.Comment{
		{On: "table", Name: "notes", Value: "old"},
		{On: "column", Table: "notes", Name: "id", Value: "id"},
		{On: "view", Name: "v", Value: "not diffed"},
	}
	updated := project(schema("public", []pgd.Table{tbl}, nil))
	updated.Comments = []pgd.Comment{
		{On: "table", Schema: "public", Name: "notes", Value: "new"},
		{On: "column", Table: "notes", Name: "body", Value: "it's text"},
	}

	r := Diff(old, updated)
	sqls := changeSQLs(r)
	assert.Equal(t, []string{
		`COMMENT ON TABLE "notes" IS 'new';`,
		`COMMENT ON COLUMN "notes"."body" IS 'it''s text';`,
		`COMMENT ON COLUMN "notes"."id" IS NULL;`,
	}, sqls)

	// the section overrides the attribute, as in generated DDL
	moved := project(schema("public", []pgd.Table{{Name: "notes", Comment: "new", Columns: tbl.Columns}}, nil))
	moved.Comments = updated.Comments[1:]
	assert.Empty(t, Diff(updated, moved).Changes)
}

func TestDiff_DroppedObjectComment(t *testing.T) {
	old := project(schema("public", []pgd.Table{{Name: "t", Comment: "table", Columns: []pgd.Column{col("id", "integer"), {Name: "x", Type: "text", Comment: "x"}}}}, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Comment: "table", Columns: []pgd.Column{col("id", "integer")}}}, nil))

	r := Diff(old, updated)
	require.Len(t, r.Changes, 1)
	assert.Contains(t, r.Changes[0].SQL, `DROP COLUMN "x"`)
}

func TestDiff_Generated(t *testing.T) {
	gen := func(expr, stored string) pgd.Column {
		return pgd.Column{Name: "full", Type: "text", Comment: "c", Generated: &pgd.Generated{Expression: expr, Stored: stored}}
	}
	plain := pgd.Column{Name: "full", Type: "text", Comment: "c"}

	tests := []struct {
		name      string
		old, new  pgd.Column
		want      []string
		recreated bool
	}{
		{"stored expression", gen("a || b", "true"), gen("b || a", "true"),
			[]string{`ALTER TABLE "t" ALTER COLUMN "full" SET EXPRESSION AS (b || a);`}, false},
		{"virtual expression", gen("a || b", "false"), gen("b || a", "false"),
			[]string{`ALTER TABLE "t" ALTER COLUMN "full" SET EXPRESSION AS (b || a);`}, false},
		{"stored to regular", gen("a || b", "true"), plain,
			[]string{`ALTER TABLE "t" ALTER COLUMN "full" DROP EXPRESSION;`}, false},
		{"regular to stored", plain, gen("a || b", "true"),
			[]string{"ALTER TABLE \"t\" DROP COLUMN \"full\";\nALTER TABLE \"t\" ADD COLUMN \"full\" text GENERATED ALWAYS AS (a || b) STORED;", `COMMENT ON COLUMN "t"."full" IS 'c';`}, true},
		{"stored to virtual", gen("a || b", "true"), gen("a || b", "false"),
			[]string{"ALTER TABLE \"t\" DROP COLUMN \"full\";\nALTER TABLE \"t\" ADD COLUMN \"full\" text GENERATED ALWAYS AS (a || b) VIRTUAL;", `COMMENT ON COLUMN "t"."full" IS 'c';`}, true},
		{"virtual to regular", gen("a || b", "false"), plain,
			[]string{"ALTER TABLE \"t\" DROP COLUMN \"full\";\nALTER TABLE \"t\" ADD COLUMN \"full\" text;", `COMMENT ON COLUMN "t"."full" IS 'c';`}, true},
		{"unchanged", gen("a || b", "true"), gen(" a || b ", "true"), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("a", "text"), col("b", "text"), tt.old}}}, nil))
			updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("a", "text"), col("b", "text"), tt.new}}}, nil))

			r := Diff(old, updated)
			sqls := changeSQLs(r)
			assert.Equal(t, tt.want, sqls)
			assert.Equal(t, tt.recreated, len(r.Errors()) > 0)
		})
	}
}

func TestDiff_RecreatedColumnDependents(t *testing.T) {
	plain := pgd.Column{Name: "tsv", Type: "tsvector"}
	gen := pgd.Column{Name: "tsv", Type: "tsvector", Generated: &pgd.Generated{Expression: "to_tsvector('simple', body)", Stored: "true"}}
	table := func(c pgd.Column) pgd.Table {
		return pgd.Table{
			Name:    "docs",
			Columns: []pgd.Column{colNN("id", "integer"), col("body", "text"), c},
			Uniques: []pgd.Unique{{Name: "uq_docs_tsv", Columns: []pgd.ColRef{{Name: "tsv"}}}},
			Checks:  []pgd.Check{{Name: "chk_docs_tsv", Expression: `"tsv" IS NOT NULL`}, {Name: "chk_docs_id", Expression: "id > 0"}},
		}
	}
	indexes := []pgd.Index{
		{Name: "ix_docs_tsv", Table: "docs", Using: "gin", Keys: []pgd.IndexKey{{Name: "tsv"}}},
		{Name: "ix_docs_len", Table: "docs", Keys: []pgd.IndexKey{{Expression: "length(tsv)"}}},
		{Name: "ix_docs_id", Table: "docs", Keys: []pgd.IndexKey{{Name: "id"}}, Where: &pgd.WhereClause{Value: "tsvector_ok(id)"}},
	}
	old := project(schema("public", []pgd.Table{table(plain)}, indexes))
	updated := project(schema("public", []pgd.Table{table(gen)}, indexes))

	sql := Diff(old, updated).SQL()
	assert.Contains(t, sql, `DROP COLUMN "tsv"`)
	assert.Contains(t, sql, `CREATE INDEX "ix_docs_tsv"`)
	assert.Contains(t, sql, `CREATE INDEX "ix_docs_len"`)
	assert.NotContains(t, sql, `"ix_docs_id"`)
	assert.Contains(t, sql, `ADD CONSTRAINT "uq_docs_tsv" UNIQUE`)
	assert.Contains(t, sql, `ADD CONSTRAINT "chk_docs_tsv" CHECK`)
	assert.NotContains(t, sql, `"chk_docs_id"`)
}

func TestMentionsColumn(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{"lower(name)", true},
		{`lower("name")`, true},
		{"name IS NOT NULL", true},
		{"t.name = 1", true},
		{"first_name || ' '", false},
		{"names", false},
		{"name(x)", false},
		{`"Name"`, false},
		{"kind <> 'name'", false},
		{"kind <> 'it''s name' AND name > 0", true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			assert.Equal(t, tt.want, mentionsColumn(tt.expr, "name"))
		})
	}
}

func TestDiff_Enums(t *testing.T) {
	enum := func(labels ...string) *pgd.Project {
		return &pgd.Project{Types: &pgd.Types{Enums: []pgd.Enum{{Name: "st", Labels: labels}}}}
	}
	tests := []struct {
		name    string
		old     *pgd.Project
		updated *pgd.Project
		want    []string
		hazard  bool
	}{
		{"append", enum("a", "b"), enum("a", "b", "c", "d"),
			[]string{`ALTER TYPE "st" ADD VALUE 'c';`, `ALTER TYPE "st" ADD VALUE 'd';`}, false},
		{"insert in the middle", enum("active", "inactive", "deleted"), enum("active", "pending", "inactive", "deleted"),
			[]string{`ALTER TYPE "st" ADD VALUE 'pending' AFTER 'active';`}, false},
		{"insert first", enum("a", "b"), enum("x", "y", "a", "b"),
			[]string{`ALTER TYPE "st" ADD VALUE 'x' BEFORE 'a';`, `ALTER TYPE "st" ADD VALUE 'y' AFTER 'x';`}, false},
		{"quote", enum("a"), enum("it's", "a"),
			[]string{`ALTER TYPE "st" ADD VALUE 'it''s' BEFORE 'a';`}, false},
		{"reorder", enum("a", "b"), enum("b", "a"),
			[]string{`-- Cannot reorder values of enum "st"; requires type recreation`}, true},
		{"drop value", enum("a", "b"), enum("a"),
			[]string{`-- Cannot drop value 'b' of enum "st"; requires type recreation`}, true},
		{"new enum", &pgd.Project{}, enum("a", "b"),
			[]string{`CREATE TYPE "st" AS ENUM ('a', 'b');`}, false},
		{"dropped enum", enum("a", "b"), &pgd.Project{},
			[]string{`DROP TYPE "st";`}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Diff(tt.old, tt.updated)
			sqls := changeSQLs(r)
			assert.Equal(t, tt.want, sqls)
			assert.Equal(t, tt.hazard, len(r.Errors()) > 0)
			for _, c := range r.Changes {
				if strings.Contains(c.SQL, "ADD VALUE") {
					require.Len(t, c.Hazards, 1)
					assert.Equal(t, "ENUM_ADD_VALUE", c.Hazards[0].Code)
				}
			}
		})
	}
}

func TestDiff_EnumBeforeTable(t *testing.T) {
	old := project(schema("public", nil, nil))
	updated := project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("s", "st")}}}, nil))
	updated.Types = &pgd.Types{Enums: []pgd.Enum{{Name: "st", Labels: []string{"a"}}}}

	r := Diff(old, updated)
	require.Len(t, r.Changes, 2)
	assert.Equal(t, "enum", r.Changes[0].Object)
	assert.Equal(t, "table", r.Changes[1].Object)
}

func TestDiff_IndexChanges(t *testing.T) {
	base := pgd.Index{Name: "ix", Table: "t", Using: "gin", Keys: []pgd.IndexKey{{Expression: "(params ->> 'mnn')", Opclass: "gin_trgm_ops"}}}
	with := func(f func(*pgd.Index)) pgd.Index {
		i := base
		i.Keys = slices.Clone(base.Keys)
		f(&i)
		return i
	}
	tests := []struct {
		name    string
		updated pgd.Index
		changed bool
	}{
		{"same", with(func(*pgd.Index) {}), false},
		{"expression", with(func(i *pgd.Index) { i.Keys[0].Expression = "(params ->> 'inn')" }), true},
		{"expression opclass", with(func(i *pgd.Index) { i.Keys[0].Opclass = "gin_trgm_ops2" }), true},
		{"expression order", with(func(i *pgd.Index) { i.Keys[0].Order = "desc" }), true},
		{"expression nulls", with(func(i *pgd.Index) { i.Keys[0].Nulls = "last" }), true},
		{"include", with(func(i *pgd.Index) { i.Include = &pgd.Include{Columns: []pgd.ColRef{{Name: "id"}}} }), true},
		{"nulls not distinct", with(func(i *pgd.Index) { i.NullsDistinct = "false" }), true},
		{"with", with(func(i *pgd.Index) { i.With = &pgd.With{Params: []pgd.WithParam{{Name: "fastupdate", Value: "off"}}} }), true},
		{"tablespace", with(func(i *pgd.Index) { i.Tablespace = "fast" }), true},
		{"concurrently is a hint", with(func(i *pgd.Index) { i.Concurrently = "true" }), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Diff(project(schema("public", nil, []pgd.Index{base})), project(schema("public", nil, []pgd.Index{tt.updated})))
			if !tt.changed {
				assert.Empty(t, r.Changes)
				return
			}
			require.Len(t, r.Changes, 2)
			assert.Equal(t, `DROP INDEX "ix";`, r.Changes[0].SQL)
			assert.Contains(t, r.Changes[1].SQL, `CREATE INDEX "ix"`)
		})
	}
}

func TestDiff_IndexDefaultsAreEqual(t *testing.T) {
	old := pgd.Index{Name: "ix", Table: "t", Keys: []pgd.IndexKey{{Name: "a"}, {Expression: "lower(b)"}}}
	// pg_get_indexdef wraps expressions in parentheses, the SQL converter doesn't
	updated := pgd.Index{Name: "ix", Table: "t", Using: "btree", Unique: "false", NullsDistinct: "true",
		Keys: []pgd.IndexKey{{Name: "a", Order: "asc"}, {Expression: "(lower(b))"}}}
	assert.Empty(t, Diff(project(schema("public", nil, []pgd.Index{old})), project(schema("public", nil, []pgd.Index{updated}))).Changes)
}

func TestDiff_IndexRename(t *testing.T) {
	expr := func(name, value string) pgd.Index {
		return pgd.Index{Name: name, Table: "t", Keys: []pgd.IndexKey{{Expression: value}}}
	}

	// renamed index: same definition under a new name is not re-created
	r := Diff(project(schema("public", nil, []pgd.Index{expr("ix_old", "lower(a)")})),
		project(schema("public", nil, []pgd.Index{expr("ix_new", "lower(a)")})))
	assert.Empty(t, r.Changes)

	// indexes on different expressions are different indexes
	r = Diff(project(schema("public", nil, []pgd.Index{expr("ix_a", "lower(a)")})),
		project(schema("public", nil, []pgd.Index{expr("ix_b", "lower(b)")})))
	require.Len(t, r.Changes, 2)
	assert.Equal(t, `DROP INDEX "ix_a";`, r.Changes[0].SQL)
	assert.Contains(t, r.Changes[1].SQL, `CREATE INDEX "ix_b"`)

	// a rename candidate must not steal an index matched by name
	r = Diff(project(schema("public", nil, []pgd.Index{idx("ix_a", "t", "x")})),
		project(schema("public", nil, []pgd.Index{idx("ix_a", "t", "y"), idx("ix_b", "t", "x")})))
	var sqls []string
	for _, c := range r.Changes {
		sqls = append(sqls, strings.SplitN(c.SQL, " (", 2)[0])
	}
	assert.Equal(t, []string{`DROP INDEX "ix_a";`, `CREATE INDEX "ix_a" ON "t"`, `CREATE INDEX "ix_b" ON "t"`}, sqls)
}

func TestDiff_FKRenameCandidates(t *testing.T) {
	tbl := func(fks ...pgd.ForeignKey) []pgd.Table {
		return []pgd.Table{{Name: "t", Columns: []pgd.Column{col("x", "integer"), col("y", "integer")}, FKs: fks}}
	}
	r := Diff(project(schema("public", tbl(fk("fk_a", "p", "restrict", "x", "id")), nil)),
		project(schema("public", tbl(fk("fk_a", "p", "restrict", "y", "id"), fk("fk_b", "p", "restrict", "x", "id")), nil)))

	var names []string
	for _, c := range r.Changes {
		names = append(names, c.Action+":"+c.Name)
	}
	assert.Equal(t, []string{"drop:fk_a", "add:fk_a", "add:fk_b"}, names)
}

func TestDiff_SchemaQualifiedIndexDrop(t *testing.T) {
	r := Diff(project(schema("app", nil, []pgd.Index{idx("ix", "t", "a")})), project(schema("app", nil, nil)))
	require.Len(t, r.Changes, 1)
	assert.Equal(t, `DROP INDEX "app"."ix";`, r.Changes[0].SQL)
}

func TestDiff_DropTablesInDependencyOrder(t *testing.T) {
	keep := pgd.Table{Name: "users", Columns: []pgd.Column{col("id", "integer")}}
	parent := pgd.Table{Name: "albums", Columns: []pgd.Column{col("id", "integer")}}
	child := pgd.Table{Name: "photos", Columns: []pgd.Column{col("albumId", "integer")},
		FKs: []pgd.ForeignKey{fk("fk_photos_albumId", "public.albums", "restrict", "albumId", "id")}}
	a := pgd.Table{Name: "a", Columns: []pgd.Column{col("bId", "integer")}, FKs: []pgd.ForeignKey{fk("fk_a_bId", "b", "restrict", "bId", "id")}}
	b := pgd.Table{Name: "b", Columns: []pgd.Column{col("aId", "integer")}, FKs: []pgd.ForeignKey{fk("fk_b_aId", "a", "restrict", "aId", "id"), fk("fk_b_self", "b", "restrict", "aId", "id")}}

	r := Diff(project(schema("public", []pgd.Table{keep, parent, child}, nil)), project(schema("public", []pgd.Table{keep}, nil)))
	assert.Equal(t, []string{`DROP TABLE "photos";`, `DROP TABLE "albums";`}, changeSQLs(r))

	// FK cycle: drop the FKs between the tables first
	r = Diff(project(schema("public", []pgd.Table{keep, a, b}, nil)), project(schema("public", []pgd.Table{keep}, nil)))
	assert.Equal(t, []string{
		`ALTER TABLE "a" DROP CONSTRAINT "fk_a_bId";`,
		`ALTER TABLE "b" DROP CONSTRAINT "fk_b_aId";`,
		`DROP TABLE "a";`,
		`DROP TABLE "b";`,
	}, changeSQLs(r))
}

func TestDiff_ChangedCheckAndUnique(t *testing.T) {
	table := func(check string, uq pgd.Unique) []pgd.Table {
		return []pgd.Table{{
			Name:    "t",
			Columns: []pgd.Column{col("a", "text"), col("b", "text")},
			Checks:  []pgd.Check{{Name: "chk_t_a", Expression: check}},
			Uniques: []pgd.Unique{uq},
		}}
	}
	old := project(schema("public", table("a IN ('x', 'y')", pgd.Unique{Name: "uq_t", Columns: []pgd.ColRef{{Name: "a"}}}), nil))
	updated := project(schema("public", table("a IN ('x', 'y', 'z')", pgd.Unique{Name: "uq_t", NullsDistinct: "false", Columns: []pgd.ColRef{{Name: "a"}, {Name: "b"}}}), nil))

	sqls := changeSQLs(Diff(old, updated))
	assert.Equal(t, []string{
		`ALTER TABLE "t" DROP CONSTRAINT "chk_t_a";`,
		`ALTER TABLE "t" DROP CONSTRAINT "uq_t";`,
		`ALTER TABLE "t" ADD CONSTRAINT "uq_t" UNIQUE NULLS NOT DISTINCT("a", "b");`,
		`ALTER TABLE "t" ADD CONSTRAINT "chk_t_a" CHECK (a IN ('x', 'y', 'z'));`,
	}, sqls)

	assert.Empty(t, Diff(old, old).Changes)
}

func TestDiff_IndexKeyOrder(t *testing.T) {
	keys := func(k ...pgd.IndexKey) []pgd.Index {
		return []pgd.Index{{Name: "ix", Table: "t", Keys: k}}
	}
	col, expr := pgd.IndexKey{Name: "a"}, pgd.IndexKey{Expression: "lower(b)"}

	r := Diff(project(schema("public", nil, keys(col, expr))), project(schema("public", nil, keys(expr, col))))
	require.Len(t, r.Changes, 2)
	assert.Equal(t, `DROP INDEX "ix";`, r.Changes[0].SQL)
	assert.Contains(t, r.Changes[1].SQL, "(lower(b)),\n\t\"a\"")

	assert.Empty(t, Diff(project(schema("public", nil, keys(col, expr))), project(schema("public", nil, keys(col, expr)))).Changes)
}

func TestDiff_UnnamedConstraints(t *testing.T) {
	// inline CHECK, UNIQUE and REFERENCES parsed from SQL have no names
	table := func(checks []pgd.Check, uniques []pgd.Unique, fks []pgd.ForeignKey) pgd.Table {
		return pgd.Table{Name: "c", Columns: []pgd.Column{col("age", "int"), col("email", "text"), col("pid", "int")},
			Checks: checks, Uniques: uniques, FKs: fks}
	}
	checks := []pgd.Check{{Expression: "age > 0"}, {Expression: "email <> ''"}}
	uniques := []pgd.Unique{{Columns: []pgd.ColRef{{Name: "email"}}}, {Columns: []pgd.ColRef{{Name: "pid"}}}}
	fks := []pgd.ForeignKey{fk("", "p", "restrict", "pid", "id"), fk("", "q", "restrict", "pid", "id")}
	same := project(schema("public", []pgd.Table{table(checks, uniques, fks)}, nil))
	assert.Empty(t, Diff(same, same).Changes)

	// added unnamed constraints are added without a name, removed ones are reported
	updated := project(schema("public", []pgd.Table{table(
		[]pgd.Check{checks[0], {Expression: "age < 200"}}, uniques[:1], fks[:1],
	)}, nil))
	r := Diff(same, updated)
	assert.Equal(t, []string{
		`-- Cannot drop unnamed FOREIGN KEY (pid) to q on "c": name it in the old model`,
		`-- Cannot drop unnamed CHECK (email <> '') on "c": name it in the old model`,
		`-- Cannot drop unnamed UNIQUE ("pid") on "c": name it in the old model`,
		`ALTER TABLE "c" ADD CHECK (age < 200);`,
	}, changeSQLs(r))
	assert.Empty(t, r.Errors(), "unnamed constraints are warnings")
}

func TestDiff_DropOrder(t *testing.T) {
	// FKs of all tables are dropped before the keys they reference
	a := func(uqCols ...string) pgd.Table {
		u := pgd.Unique{Name: "uq_a"}
		for _, c := range uqCols {
			u.Columns = append(u.Columns, pgd.ColRef{Name: c})
		}
		return pgd.Table{Name: "a", Columns: []pgd.Column{col("x", "int"), col("y", "int")}, Uniques: []pgd.Unique{u}}
	}
	b := func(cols ...string) pgd.Table {
		f := fk("fk_b_a", "a", "restrict")
		for _, c := range cols {
			f.Columns = append(f.Columns, pgd.FKCol{Name: c, References: c})
		}
		return pgd.Table{Name: "b", Columns: []pgd.Column{col("x", "int"), col("y", "int")}, FKs: []pgd.ForeignKey{f}}
	}
	r := Diff(project(schema("public", []pgd.Table{a("x"), b("x")}, nil)), project(schema("public", []pgd.Table{a("x", "y"), b("x", "y")}, nil)))
	sqls := changeSQLs(r)
	require.GreaterOrEqual(t, len(sqls), 2)
	assert.Equal(t, `ALTER TABLE "b" DROP CONSTRAINT "fk_b_a";`, sqls[0])
	assert.Equal(t, `ALTER TABLE "a" DROP CONSTRAINT "uq_a";`, sqls[1])

	// tables of different schemas: the referencing table goes first
	t1 := pgd.Table{Name: "t1", Columns: []pgd.Column{col("id", "int")}}
	t2 := pgd.Table{Name: "t2", Columns: []pgd.Column{col("id", "int")}, FKs: []pgd.ForeignKey{fk("fk_t2_t1", "a.t1", "restrict", "id", "id")}}
	old := project(schema("a", []pgd.Table{t1}, nil), schema("b", []pgd.Table{t2}, nil))
	r = Diff(old, project(schema("a", nil, nil), schema("b", nil, nil)))
	assert.Equal(t, []string{`DROP TABLE "b"."t2";`, `DROP TABLE "a"."t1";`}, changeSQLs(r))
}

func TestDiff_NewSchema(t *testing.T) {
	r := Diff(project(schema("public", nil, nil)),
		project(schema("public", nil, nil), schema("app", []pgd.Table{{Name: "t", Columns: []pgd.Column{col("id", "int")}}}, nil)))
	sqls := changeSQLs(r)
	require.Len(t, sqls, 2)
	assert.Equal(t, `CREATE SCHEMA IF NOT EXISTS "app";`, sqls[0])
	assert.Contains(t, sqls[1], `CREATE TABLE "app"."t"`)
}

func TestDiff_NamesCantBreakOutOfComments(t *testing.T) {
	// a label or name with a line break must not turn the rest of a comment into SQL
	old := &pgd.Project{Types: &pgd.Types{Enums: []pgd.Enum{{Name: "st", Labels: []string{"a", "x\nDROP TABLE victims; --"}}}}}
	updated := &pgd.Project{Types: &pgd.Types{Enums: []pgd.Enum{{Name: "st", Labels: []string{"a"}}}}}
	updated.Schemas = []pgd.Schema{{Name: "public"}}
	old.Schemas = []pgd.Schema{{Name: "public", Tables: []pgd.Table{{Name: "t\nDROP TABLE victims2; --", Columns: []pgd.Column{col("id", "int")}}}}}

	sql := Diff(old, updated).SQL()
	tree, err := pgquery.Parse(sql)
	require.NoError(t, err, sql)
	require.Len(t, tree.Stmts, 1, sql) // only DROP TABLE "t…"; the enum change is a comment
	assert.Equal(t, "t\nDROP TABLE victims2; --", tree.Stmts[0].Stmt.GetDropStmt().Objects[0].GetList().Items[0].GetString_().Sval)
}

func TestDiff_ExpressionFormatting(t *testing.T) {
	// the same schema as introspected from a database (pg_get_expr) and as written in DDL
	table := func(check, generated, index, where string) *pgd.Project {
		return project(schema("public", []pgd.Table{{
			Name: "t",
			Columns: []pgd.Column{col("price", "numeric"), col("qty", "int"), col("code", "text"),
				{Name: "total", Type: "numeric", Generated: &pgd.Generated{Expression: generated, Stored: "true"}}},
			Checks: []pgd.Check{{Name: "chk_t_price", Expression: check}},
		}}, []pgd.Index{{Name: "ix_t_code", Table: "t", Keys: []pgd.IndexKey{{Expression: index}}, Where: &pgd.WhereClause{Value: where}}}))
	}
	db := table("(price > (0)::numeric)", "(price * (qty)::numeric)", "(lower(code))", "(code <> 'x'::text)")
	ddl := table("price > 0::numeric", "price*qty::numeric", "lower(code)", "code <> 'x'")
	assert.Empty(t, changeSQLs(Diff(db, ddl)))

	changed := table("price >= 0::numeric", "price*qty::numeric", "upper(code)", "code <> 'x'")
	assert.Len(t, Diff(db, changed).Changes, 4) // CHECK and index: drop + add
}

func TestDiff_DefaultFormatting(t *testing.T) {
	table := func(def string) *pgd.Project {
		return project(schema("public", []pgd.Table{{Name: "t", Columns: []pgd.Column{colDefault("status", "varchar", def)}}}, nil))
	}
	assert.Empty(t, Diff(table("'open'::character varying"), table("'open'")).Changes)
	assert.Len(t, Diff(table("'open'::character varying"), table("'closed'")).Changes, 1)
	assert.Len(t, Diff(table("'open'"), table("")).Changes, 1)
}

func TestDiff_TableOptions(t *testing.T) {
	table := func(f func(*pgd.Table)) *pgd.Project {
		tbl := pgd.Table{Name: "t", Columns: []pgd.Column{col("id", "int")},
			With: &pgd.With{Params: []pgd.WithParam{{Name: "fillfactor", Value: "80"}, {Name: "autovacuum_enabled", Value: "false"}}}}
		f(&tbl)
		return project(schema("public", []pgd.Table{tbl}, nil))
	}
	old := table(func(*pgd.Table) {})
	tests := []struct {
		name    string
		updated *pgd.Project
		want    []string
	}{
		{"same", table(func(*pgd.Table) {}), nil},
		{"changed and removed params", table(func(t *pgd.Table) {
			t.With = &pgd.With{Params: []pgd.WithParam{{Name: "fillfactor", Value: "50"}, {Name: "toast.autovacuum_enabled", Value: "true"}}}
		}), []string{
			`ALTER TABLE "t" SET (fillfactor='50', toast.autovacuum_enabled='true');`,
			`ALTER TABLE "t" RESET (autovacuum_enabled);`,
		}},
		{"no params", table(func(t *pgd.Table) { t.With = nil }), []string{`ALTER TABLE "t" RESET (fillfactor, autovacuum_enabled);`}},
		{"unlogged", table(func(t *pgd.Table) { t.Unlogged = "true" }), []string{`ALTER TABLE "t" SET UNLOGGED;`}},
		{"tablespace", table(func(t *pgd.Table) { t.Tablespace = "fast" }), []string{`ALTER TABLE "t" SET TABLESPACE "fast";`}},
		{"default access method", table(func(t *pgd.Table) { t.Using = "heap" }), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, changeSQLs(Diff(old, tt.updated)))
		})
	}

	// back to default tablespace
	assert.Equal(t, []string{`ALTER TABLE "t" SET TABLESPACE "pg_default";`},
		changeSQLs(Diff(table(func(t *pgd.Table) { t.Tablespace = "fast" }), old)))
}

func TestDiff_NewSchemaBeforeEnum(t *testing.T) {
	updated := project(schema("public", nil, nil))
	updated.Types = &pgd.Types{Enums: []pgd.Enum{{Name: "st", Schema: "mp", Labels: []string{"a"}}}}
	assert.Equal(t, []string{`CREATE SCHEMA IF NOT EXISTS "mp";`, `CREATE TYPE "mp"."st" AS ENUM ('a');`},
		changeSQLs(Diff(project(schema("public", nil, nil)), updated)))
}

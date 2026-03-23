package diff

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

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

func idx(name, table string, cols ...string) pgd.Index {
	i := pgd.Index{Name: name, Table: table}
	for _, c := range cols {
		i.Columns = append(i.Columns, pgd.ColRef{Name: c})
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
	assert.Contains(t, r.Changes[0].SQL, `ADD CONSTRAINT "uq_email" UNIQUE ("email")`)
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
	oldIdx := pgd.Index{Name: "idx_x", Table: "t", Unique: "true", Columns: []pgd.ColRef{{Name: "workspaceId"}, {Name: "statusId"}}}
	newIdx := pgd.Index{Name: "idx_x", Table: "t", Unique: "true", Columns: []pgd.ColRef{{Name: "workspaceId"}}, Where: &pgd.WhereClause{Value: `"statusId" IN (1, 2)`}}

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

package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func testProject(tables ...pgd.Table) *pgd.Project {
	return &pgd.Project{
		Schemas: []pgd.Schema{{Name: "public", Tables: tables}},
	}
}

func TestFix_ColumnTypeReplacements(t *testing.T) {
	tests := []struct {
		code    string
		colType string
		length  int
		want    string
	}{
		{RulePreferText, "char", 50, "text"},
		{RuleAvoidMoney, "money", 0, "numeric"},
		{RulePreferTSTZ, "timestamp", 0, "timestamptz"},
		{RuleAvoidTimeTZ, "timetz", 0, "time"},
		{RulePreferJsonb, "json", 0, "jsonb"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			p := testProject(pgd.Table{
				Name:    "users",
				Columns: []pgd.Column{{Name: "col1", Type: tt.colType, Length: tt.length}},
			})
			issues := []Issue{{Code: tt.code, Path: "public.users.col1"}}
			results := Fix(p, issues)
			require.Len(t, results, 1)
			assert.Equal(t, tt.code, results[0].Code)
			assert.Equal(t, tt.want, p.Schemas[0].Tables[0].Columns[0].Type)
			assert.Equal(t, 0, p.Schemas[0].Tables[0].Columns[0].Length)
		})
	}
}

func TestFix_PreferIdentity(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "serial"}},
	})
	results := Fix(p, []Issue{{Code: RulePreferIdentity, Path: "public.users.id"}})
	require.Len(t, results, 1)
	col := p.Schemas[0].Tables[0].Columns[0]
	assert.Equal(t, "integer", col.Type)
	assert.NotNil(t, col.Identity)
	assert.Equal(t, "by-default", col.Identity.Generated)
}

func TestFix_PreferIdentity_Bigserial(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "events",
		Columns: []pgd.Column{{Name: "id", Type: "bigserial"}},
	})
	results := Fix(p, []Issue{{Code: RulePreferIdentity, Path: "public.events.id"}})
	require.Len(t, results, 1)
	assert.Equal(t, "bigint", p.Schemas[0].Tables[0].Columns[0].Type)
}

func TestFix_ClearDefault(t *testing.T) {
	p := testProject(pgd.Table{
		Name: "users",
		Columns: []pgd.Column{{
			Name:     "id",
			Type:     "integer",
			Default:  "0",
			Identity: &pgd.Identity{Generated: "always"},
		}},
	})
	results := Fix(p, []Issue{{Code: RuleIdentityDefault, Path: "public.users.id"}})
	require.Len(t, results, 1)
	assert.Empty(t, p.Schemas[0].Tables[0].Columns[0].Default)
}

func TestFix_PKNullable(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "integer", Nullable: "true"}},
		PK:      &pgd.PrimaryKey{Name: "pk_users", Columns: []pgd.ColRef{{Name: "id"}}},
	})
	results := Fix(p, []Issue{{Code: RulePKNullable, Path: "public.users.id"}})
	require.Len(t, results, 1)
	assert.Equal(t, "false", p.Schemas[0].Tables[0].Columns[0].Nullable)
}

func TestFix_MissingFKIndex(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "user_id", Type: "integer"}},
		FKs: []pgd.ForeignKey{{
			Name:     "fk_orders_user_id",
			ToTable:  "users",
			OnDelete: "CASCADE",
			Columns:  []pgd.FKCol{{Name: "user_id", References: "id"}},
		}},
	})

	issue := Issue{
		Code:    RuleMissingFKIndex,
		Path:    "public.orders",
		Message: `FK "fk_orders_user_id" columns have no matching index`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "ix_orders_user_id")

	// Verify index was added
	require.Len(t, p.Schemas[0].Indexes, 1)
	idx := p.Schemas[0].Indexes[0]
	assert.Equal(t, "ix_orders_user_id", idx.Name)
	assert.Equal(t, "orders", idx.Table)
	assert.Equal(t, "btree", idx.Using)
	require.Len(t, idx.Columns, 1)
	assert.Equal(t, "user_id", idx.Columns[0].Name)
}

func TestFix_MissingFKIndex_MultiColumn(t *testing.T) {
	p := testProject(pgd.Table{
		Name: "order_items",
		Columns: []pgd.Column{
			{Name: "order_id", Type: "integer"},
			{Name: "product_id", Type: "integer"},
		},
		FKs: []pgd.ForeignKey{{
			Name:    "fk_order_items_order_product",
			ToTable: "orders",
			Columns: []pgd.FKCol{{Name: "order_id", References: "id"}, {Name: "product_id", References: "id"}},
		}},
	})

	issue := Issue{
		Code:    RuleMissingFKIndex,
		Path:    "public.order_items",
		Message: `FK "fk_order_items_order_product" columns have no matching index`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)

	require.Len(t, p.Schemas[0].Indexes, 1)
	idx := p.Schemas[0].Indexes[0]
	assert.Equal(t, "ix_order_items_order_id_product_id", idx.Name)
	require.Len(t, idx.Columns, 2)
}

func TestFix_NoPK_SnakeCase(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "email", Type: "text"}},
	})
	p.ProjectMeta.Settings.Naming.Convention = "snake_case"

	results := Fix(p, []Issue{{Code: RuleNoPK, Path: "public.users"}})
	require.Len(t, results, 1)

	table := p.Schemas[0].Tables[0]
	// PK column prepended
	require.Len(t, table.Columns, 2)
	assert.Equal(t, "user_id", table.Columns[0].Name)
	assert.Equal(t, "integer", table.Columns[0].Type)
	assert.Equal(t, "false", table.Columns[0].Nullable)
	assert.NotNil(t, table.Columns[0].Identity)
	assert.Equal(t, "by-default", table.Columns[0].Identity.Generated)
	// Original column still there
	assert.Equal(t, "email", table.Columns[1].Name)
	// PK constraint
	require.NotNil(t, table.PK)
	assert.Equal(t, "pk_users", table.PK.Name)
	require.Len(t, table.PK.Columns, 1)
	assert.Equal(t, "user_id", table.PK.Columns[0].Name)
}

func TestFix_NoPK_CamelCase(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "email", Type: "text"}},
	})
	p.ProjectMeta.Settings.Naming.Convention = "camelCase"

	results := Fix(p, []Issue{{Code: RuleNoPK, Path: "public.users"}})
	require.Len(t, results, 1)
	assert.Equal(t, "userId", p.Schemas[0].Tables[0].Columns[0].Name)
}

func TestFix_NoPK_NoNaming(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "logs",
		Columns: []pgd.Column{{Name: "payload", Type: "jsonb"}},
	})

	results := Fix(p, []Issue{{Code: RuleNoPK, Path: "public.logs"}})
	require.Len(t, results, 1)
	// No naming convention set, but ExpectedPKName still returns singularized camelCase
	col := p.Schemas[0].Tables[0].Columns[0]
	assert.Equal(t, "logId", col.Name)
}

func TestFix_NoPK_ColumnConflict(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "items",
		Columns: []pgd.Column{{Name: "itemId", Type: "text"}},
	})

	results := Fix(p, []Issue{{Code: RuleNoPK, Path: "public.items"}})
	assert.Empty(t, results, "should skip if PK column name conflicts")
}

func TestFix_NoPK_AlreadyHasPK(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
		PK:      &pgd.PrimaryKey{Name: "pk_users", Columns: []pgd.ColRef{{Name: "id"}}},
	})

	results := Fix(p, []Issue{{Code: RuleNoPK, Path: "public.users"}})
	assert.Empty(t, results, "should skip if table already has PK")
}

func TestFix_FKNoAction(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "user_id", Type: "integer"}},
		FKs: []pgd.ForeignKey{{
			Name:     "fk_orders_user",
			ToTable:  "users",
			OnDelete: "NO ACTION",
			OnUpdate: "NO ACTION",
			Columns:  []pgd.FKCol{{Name: "user_id", References: "id"}},
		}},
	})
	issue := Issue{
		Code:    RuleFKNoAction,
		Path:    "public.orders",
		Message: `FK "fk_orders_user" uses NO ACTION — specify action explicitly`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Equal(t, "RESTRICT", p.Schemas[0].Tables[0].FKs[0].OnDelete)
	assert.Equal(t, "RESTRICT", p.Schemas[0].Tables[0].FKs[0].OnUpdate)
}

func TestFix_FKNoAction_OnlyDelete(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "user_id", Type: "integer"}},
		FKs: []pgd.ForeignKey{{
			Name:     "fk_orders_user",
			ToTable:  "users",
			OnDelete: "NO ACTION",
			OnUpdate: "CASCADE",
			Columns:  []pgd.FKCol{{Name: "user_id", References: "id"}},
		}},
	})
	issue := Issue{
		Code:    RuleFKNoAction,
		Path:    "public.orders",
		Message: `FK "fk_orders_user" uses NO ACTION — specify action explicitly`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Equal(t, "RESTRICT", p.Schemas[0].Tables[0].FKs[0].OnDelete)
	assert.Equal(t, "CASCADE", p.Schemas[0].Tables[0].FKs[0].OnUpdate)
}

func TestFix_DropDuplicateIndex(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "user_id", Type: "integer"}},
	})
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "ix_orders_user_id", Table: "orders", Columns: []pgd.ColRef{{Name: "user_id"}}},
		{Name: "ix_orders_user_id_dup", Table: "orders", Columns: []pgd.ColRef{{Name: "user_id"}}},
	}
	issue := Issue{
		Code:    RuleDupIndexCols,
		Path:    "public.ix_orders_user_id_dup",
		Message: `index has same columns as "ix_orders_user_id" on table "orders"`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Len(t, p.Schemas[0].Indexes, 1)
	assert.Equal(t, "ix_orders_user_id", p.Schemas[0].Indexes[0].Name)
}

func TestFix_DropOverlappingIndex(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "user_id", Type: "integer"}, {Name: "status", Type: "integer"}},
	})
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "ix_orders_user_id", Table: "orders", Columns: []pgd.ColRef{{Name: "user_id"}}},
		{Name: "ix_orders_user_id_status", Table: "orders", Columns: []pgd.ColRef{{Name: "user_id"}, {Name: "status"}}},
	}
	issue := Issue{
		Code:    RuleOverlapIndex,
		Path:    "public.ix_orders_user_id",
		Message: `index is a prefix of "ix_orders_user_id_status" on table "orders" — redundant`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Len(t, p.Schemas[0].Indexes, 1)
	assert.Equal(t, "ix_orders_user_id_status", p.Schemas[0].Indexes[0].Name)
}

func TestFix_DropDuplicateFK(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "user_id", Type: "integer"}},
		FKs: []pgd.ForeignKey{
			{Name: "fk_orders_user", ToTable: "users", Columns: []pgd.FKCol{{Name: "user_id", References: "id"}}},
			{Name: "fk_orders_user_dup", ToTable: "users", Columns: []pgd.FKCol{{Name: "user_id", References: "id"}}},
		},
	})
	issue := Issue{
		Code:    RuleDupFK,
		Path:    "public.orders",
		Message: `FK "fk_orders_user_dup" is a duplicate of "fk_orders_user" (same columns, same target)`,
	}
	results := Fix(p, []Issue{issue})
	require.Len(t, results, 1)
	assert.Len(t, p.Schemas[0].Tables[0].FKs, 1)
	assert.Equal(t, "fk_orders_user", p.Schemas[0].Tables[0].FKs[0].Name)
}

func TestFix_NonFixable_Skipped(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
	})
	results := Fix(p, []Issue{{Code: RuleCircularFK, Path: "public"}})
	assert.Empty(t, results)
}

func TestFix_BadPath_Skipped(t *testing.T) {
	p := testProject(pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
	})
	results := Fix(p, []Issue{{Code: RulePreferTSTZ, Path: "public.nonexistent.col"}})
	assert.Empty(t, results)
}

func TestExtractQuoted(t *testing.T) {
	assert.Equal(t, "fk_name", extractQuoted(`FK "fk_name" columns have no matching index`))
	assert.Empty(t, extractQuoted("no quotes here"))
	assert.Empty(t, extractQuoted(`only "one`))
}

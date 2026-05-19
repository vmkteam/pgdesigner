package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func testProject() *pgd.Project {
	return &pgd.Project{
		Version:       1,
		PgVersion:     "18",
		DefaultSchema: "public",
		Schemas: []pgd.Schema{{
			Name: "public",
			Tables: []pgd.Table{{
				Name: "users",
				Columns: []pgd.Column{
					{Name: "id", Type: "integer", Nullable: "false"},
					{Name: "name", Type: "text"},
				},
				PK: &pgd.PrimaryKey{Name: "pk_users", Columns: []pgd.ColRef{{Name: "id"}}},
			}},
			Indexes: []pgd.Index{
				{Name: "idx_users_name", Table: "users", Columns: []pgd.ColRef{{Name: "name"}}},
			},
		}},
	}
}

func TestProjectStore_IsDirty(t *testing.T) {
	s := NewProjectStore(testProject(), "")
	assert.False(t, s.IsDirty())

	err := s.UpdateTableColumns("users", []pgd.Column{
		{Name: "id", Type: "integer", Nullable: "false"},
		{Name: "name", Type: "text"},
		{Name: "email", Type: "text"},
	})
	require.NoError(t, err)
	assert.True(t, s.IsDirty())
}

func TestProjectStore_UpdateTableColumns(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	newCols := []pgd.Column{
		{Name: "id", Type: "bigint", Nullable: "false"},
		{Name: "email", Type: "text", Nullable: "false"},
	}
	require.NoError(t, s.UpdateTableColumns("users", newCols))
	assert.Len(t, s.Project().Schemas[0].Tables[0].Columns, 2)
	assert.Equal(t, "bigint", s.Project().Schemas[0].Tables[0].Columns[0].Type)
}

func TestProjectStore_UpdateTablePK(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	pk := &pgd.PrimaryKey{Name: "pk_new", Columns: []pgd.ColRef{{Name: "id"}, {Name: "name"}}}
	require.NoError(t, s.UpdateTablePK("users", pk))
	assert.Len(t, s.Project().Schemas[0].Tables[0].PK.Columns, 2)

	// Remove PK
	require.NoError(t, s.UpdateTablePK("users", nil))
	assert.Nil(t, s.Project().Schemas[0].Tables[0].PK)
}

func TestProjectStore_UpdateTableFKs(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	fks := []pgd.ForeignKey{{
		Name: "fk_test", ToTable: "statuses",
		Columns: []pgd.FKCol{{Name: "id", References: "id"}},
	}}
	require.NoError(t, s.UpdateTableFKs("users", fks))
	assert.Len(t, s.Project().Schemas[0].Tables[0].FKs, 1)
}

func TestProjectStore_UpdateTableIndexes(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	// Replace indexes
	newIdx := []pgd.Index{
		{Name: "idx_email", Table: "users", Columns: []pgd.ColRef{{Name: "email"}}},
	}
	require.NoError(t, s.UpdateTableIndexes("users", newIdx))

	idxs := s.Project().Schemas[0].Indexes
	assert.Len(t, idxs, 1)
	assert.Equal(t, "idx_email", idxs[0].Name)
}

func TestProjectStore_UpdateTableIndexes_PreservesPosition(t *testing.T) {
	p := &pgd.Project{
		Version: 1, DefaultSchema: "public",
		Schemas: []pgd.Schema{{
			Name: "public",
			Tables: []pgd.Table{
				{Name: "a"}, {Name: "b"}, {Name: "c"},
			},
			Indexes: []pgd.Index{
				{Name: "ia1", Table: "a"},
				{Name: "ib1", Table: "b"},
				{Name: "ib2", Table: "b"},
				{Name: "ic1", Table: "c"},
				{Name: "ia2", Table: "a"},
			},
		}},
	}

	tests := []struct {
		name   string
		table  string
		input  []pgd.Index
		expect []string
	}{
		{
			name:  "replace b-block in place, c stays after b",
			table: "b",
			input: []pgd.Index{
				{Name: "ib1", Table: "b"},
				{Name: "ib2", Table: "b"},
				{Name: "ib_new", Table: "b"},
			},
			expect: []string{"ia1", "ib1", "ib2", "ib_new", "ic1", "ia2"},
		},
		{
			name:  "add first index for table without any: append to end",
			table: "c",
			input: []pgd.Index{
				{Name: "ic1", Table: "c"},
				{Name: "ic2", Table: "c"},
			},
			expect: []string{"ia1", "ib1", "ib2", "ic1", "ic2", "ia2"},
		},
		{
			name:   "empty input drops the table's indexes, keeps others",
			table:  "a",
			input:  nil,
			expect: []string{"ib1", "ib2", "ic1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewProjectStore(deepCopyProject(p), "")
			require.NoError(t, s.UpdateTableIndexes(tc.table, tc.input))

			got := s.Project().Schemas[0].Indexes
			names := make([]string, len(got))
			for i, idx := range got {
				names[i] = idx.Name
			}
			assert.Equal(t, tc.expect, names)
		})
	}
}

func TestProjectStore_UpdateTableGeneral(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	newName := "accounts"
	comment := "User accounts"
	unlogged := true
	require.NoError(t, s.UpdateTableGeneral("users", &newName, &comment, &unlogged, nil))

	tbl := s.Project().Schemas[0].Tables[0]
	assert.Equal(t, "accounts", tbl.Name)
	assert.Equal(t, "User accounts", tbl.Comment)
	assert.Equal(t, "true", tbl.Unlogged)

	// Index references should be updated to new table name.
	assert.Equal(t, "accounts", s.Project().Schemas[0].Indexes[0].Table)
}

func TestProjectStore_RenameUpdatesFKRefs(t *testing.T) {
	p := testProject()
	// Add "orders" table with FK to "users"
	p.Schemas[0].Tables = append(p.Schemas[0].Tables, pgd.Table{
		Name:    "orders",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "userId", Type: "integer"}},
		FKs: []pgd.ForeignKey{{
			Name: "fk_orders_user", ToTable: "users",
			Columns: []pgd.FKCol{{Name: "userId", References: "id"}},
		}},
	})
	s := NewProjectStore(p, "")

	newName := "accounts"
	require.NoError(t, s.UpdateTableGeneral("users", &newName, nil, nil, nil))

	// FK in orders should now point to "accounts"
	assert.Equal(t, "accounts", s.Project().Schemas[0].Tables[1].FKs[0].ToTable)
}

func TestProjectStore_CreateTable(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	require.NoError(t, s.CreateTable("public", "orders"))
	assert.Len(t, s.Project().Schemas[0].Tables, 2)
	assert.Equal(t, "orders", s.Project().Schemas[0].Tables[1].Name)
	assert.NotNil(t, s.Project().Schemas[0].Tables[1].PK)
}

func TestProjectStore_DeleteTable(t *testing.T) {
	s := NewProjectStore(testProject(), "")

	require.NoError(t, s.DeleteTable("users"))
	assert.Empty(t, s.Project().Schemas[0].Tables)
	assert.Empty(t, s.Project().Schemas[0].Indexes) // indexes removed too
}

func TestProjectStore_Save(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pgd")

	s := NewProjectStore(testProject(), path)
	require.NoError(t, s.UpdateTableColumns("users", []pgd.Column{{Name: "id", Type: "integer"}}))
	assert.True(t, s.IsDirty())

	require.NoError(t, s.Save())
	assert.False(t, s.IsDirty())

	// File exists
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "<pgd")
}

func TestProjectStore_SaveBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pgd")

	s := NewProjectStore(testProject(), path)
	require.NoError(t, s.UpdateTableColumns("users", []pgd.Column{{Name: "id", Type: "integer"}}))

	require.NoError(t, s.SaveBackup())
	_, err := os.Stat(path + ".bak")
	assert.NoError(t, err, ".bak should exist")

	// Save removes .bak
	require.NoError(t, s.Save())
	_, err = os.Stat(path + ".bak")
	assert.True(t, os.IsNotExist(err), ".bak should be removed after Save")
}

func TestProjectStore_NotFound(t *testing.T) {
	s := NewProjectStore(testProject(), "")
	require.Error(t, s.UpdateTableColumns("nonexistent", nil))
	require.Error(t, s.UpdateTableIndexes("nonexistent", nil))
	require.Error(t, s.DeleteTable("nonexistent"))
	require.Error(t, s.CreateTable("nonexistent_schema", "t"))
}

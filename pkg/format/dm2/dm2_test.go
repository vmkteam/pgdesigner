package dm2

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	data, err := os.ReadFile("testdata/EazyPhoto.dm2")
	require.NoError(t, err)

	project, err := Convert(data, "EazyPhoto")
	require.NoError(t, err)

	assert.Equal(t, "EazyPhoto", project.ProjectMeta.Name)
	assert.Equal(t, "18", project.PgVersion)
	assert.Equal(t, "public", project.DefaultSchema)

	require.Len(t, project.Schemas, 1)
	schema := project.Schemas[0]
	assert.Equal(t, "public", schema.Name)
	assert.Len(t, schema.Tables, 15)

	// layout
	require.Len(t, project.Layouts.Layouts, 1)
	layout := project.Layouts.Layouts[0]
	assert.Equal(t, "Default Diagram", layout.Name)
	assert.Len(t, layout.Entities, 15)

	// golden file
	got, err := xml.MarshalIndent(project, "", "  ")
	require.NoError(t, err)
	got = []byte(xml.Header + string(got) + "\n")

	want, err := os.ReadFile("testdata/EazyPhoto.pgd")
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got), "golden mismatch")

	// build table map
	tableMap := make(map[string]int)
	for i, tbl := range schema.Tables {
		tableMap[tbl.Name] = i
	}

	t.Run("users", func(t *testing.T) {
		tbl := schema.Tables[tableMap["users"]]
		assert.Equal(t, "users", tbl.Name)
		require.NotNil(t, tbl.PK)
		assert.Equal(t, "userId", tbl.PK.Columns[0].Name)
		require.Len(t, tbl.Columns, 6)
		assert.Equal(t, "userId", tbl.Columns[0].Name)
		assert.Equal(t, "integer", tbl.Columns[0].Type)
		assert.Equal(t, "false", tbl.Columns[0].Nullable)
		require.NotNil(t, tbl.Columns[0].Identity)
		assert.Equal(t, "by-default", tbl.Columns[0].Identity.Generated)
	})

	t.Run("FK count", func(t *testing.T) {
		totalFKs := 0
		for _, tbl := range schema.Tables {
			totalFKs += len(tbl.FKs)
		}
		assert.Equal(t, 25, totalFKs)
	})

	t.Run("indexes", func(t *testing.T) {
		assert.Len(t, schema.Indexes, 30)
	})

	t.Run("vfsFolders FK", func(t *testing.T) {
		tbl := schema.Tables[tableMap["vfsFolders"]]
		require.GreaterOrEqual(t, len(tbl.FKs), 2)
	})
}

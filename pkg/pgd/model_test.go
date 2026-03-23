package pgd

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"chinook", "../format/sql/testdata/chinook.pgd"},
		{"northwind", "../format/sql/testdata/northwind.pgd"},
		{"pagila", "../format/sql/testdata/pagila.pgd"},
		{"airlines", "../format/sql/testdata/airlines.pgd"},
		{"adventureworks", "../format/sql/testdata/adventureworks.pgd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			// unmarshal
			var project Project
			err = xml.Unmarshal(data, &project)
			require.NoError(t, err)

			// marshal back
			got, err := xml.MarshalIndent(&project, "", "  ")
			require.NoError(t, err)
			got = []byte(xml.Header + string(got) + "\n")

			// compare
			assert.Equal(t, string(data), string(got), "round-trip mismatch for %s", tt.file)
		})
	}
}

func TestUnmarshalChinook(t *testing.T) {
	data, err := os.ReadFile("../format/sql/testdata/chinook.pgd")
	require.NoError(t, err)

	var p Project
	require.NoError(t, xml.Unmarshal(data, &p))

	assert.Equal(t, 1, p.Version)
	assert.Equal(t, "18", p.PgVersion)
	assert.Equal(t, "public", p.DefaultSchema)
	assert.Equal(t, "chinook", p.ProjectMeta.Name)

	require.Len(t, p.Schemas, 1)
	schema := p.Schemas[0]
	assert.Equal(t, "public", schema.Name)
	assert.Len(t, schema.Tables, 11)

	// album table
	album := schema.Tables[findTable(schema.Tables, "album")]
	assert.Len(t, album.Columns, 3)
	assert.Equal(t, "album_id", album.Columns[0].Name)
	assert.Equal(t, "integer", album.Columns[0].Type)
	assert.Equal(t, "false", album.Columns[0].Nullable)
	require.NotNil(t, album.PK)
	assert.Len(t, album.PK.Columns, 1)

	// customer table FK
	customer := schema.Tables[findTable(schema.Tables, "customer")]
	require.Len(t, customer.FKs, 1)
	assert.Equal(t, "employee", customer.FKs[0].ToTable)

	// indexes
	assert.Len(t, schema.Indexes, 11)

	// layouts
	require.Len(t, p.Layouts.Layouts, 1)
	assert.Equal(t, "Default Diagram", p.Layouts.Layouts[0].Name)
}

func TestUnmarshalAdventureWorks(t *testing.T) {
	data, err := os.ReadFile("../format/sql/testdata/adventureworks.pgd")
	require.NoError(t, err)

	var p Project
	require.NoError(t, xml.Unmarshal(data, &p))

	// multi-schema: humanresources, person, production, purchasing, sales
	assert.GreaterOrEqual(t, len(p.Schemas), 5)

	var tableCount, fkCount, indexCount int
	for _, s := range p.Schemas {
		tableCount += len(s.Tables)
		indexCount += len(s.Indexes)
		for _, tbl := range s.Tables {
			fkCount += len(tbl.FKs)
		}
	}
	assert.Equal(t, 68, tableCount)
	assert.Equal(t, 89, fkCount)
	assert.Equal(t, 2, indexCount)

	// domains
	require.NotNil(t, p.Types)
	assert.GreaterOrEqual(t, len(p.Types.Domains), 6)

	// views
	require.NotNil(t, p.Views)
	assert.NotEmpty(t, p.Views.Views)

	// comments
	assert.NotEmpty(t, p.Comments)
}

func findTable(tables []Table, name string) int {
	for i, t := range tables {
		if t.Name == name {
			return i
		}
	}
	return -1
}

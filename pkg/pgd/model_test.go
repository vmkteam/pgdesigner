package pgd

import (
	"encoding/xml"
	"os"
	"strings"
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
		{"synthetic", "../../docs/pgd-format/examples/synthetic.pgd"}, // index <expression> keys
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

func TestIndexKeys_XMLOrder(t *testing.T) {
	const pgdXML = `<index name="uq_personAwards" table="personAwards" unique="true">
  <column name="personId"></column>
  <expression><![CDATA[COALESCE("awardId", 0)]]></expression>
  <column name="isWinner" order="desc"></column>
  <expression opclass="text_pattern_ops"><![CDATA[lower(title)]]></expression>
</index>`
	var idx Index
	require.NoError(t, xml.Unmarshal([]byte(pgdXML), &idx))
	assert.Equal(t, []IndexKey{
		{Name: "personId"},
		{Expression: `COALESCE("awardId", 0)`},
		{Name: "isWinner", Order: "desc"},
		{Expression: "lower(title)", Opclass: "text_pattern_ops"},
	}, idx.Keys)
	assert.Equal(t, []string{"personId", "isWinner"}, idx.KeyColumns())

	// saved in the same order
	var out strings.Builder
	enc := xml.NewEncoder(&out)
	enc.Indent("", "  ")
	require.NoError(t, enc.EncodeElement(&idx, xml.StartElement{Name: xml.Name{Local: "index"}}))
	assert.Equal(t, pgdXML, out.String())

	// and generated in key order
	assert.Equal(t, "\t\"personId\",\n\t(COALESCE(\"awardId\", 0)),\n\t\"isWinner\" DESC,\n\t(lower(title)) text_pattern_ops", indexElements(&idx))
}

func TestIndexKeys_UnknownElement(t *testing.T) {
	var idx Index
	err := xml.Unmarshal([]byte(`<index name="ix" table="t"><colum name="a"></colum></index>`), &idx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected <colum> in <index>")
}

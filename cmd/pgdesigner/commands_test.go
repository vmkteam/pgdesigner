package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func diffFixture(t *testing.T) *diff.DiffResult {
	t.Helper()
	old := &pgd.Project{Version: 1, DefaultSchema: "public", Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{{
			Name:    "users",
			Columns: []pgd.Column{{Name: "id", Type: "integer", Nullable: "false"}},
			PK:      &pgd.PrimaryKey{Name: "pk_users", Columns: []pgd.ColRef{{Name: "id"}}},
		}},
	}}}
	updated := &pgd.Project{Version: 1, DefaultSchema: "public", Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{{
			Name: "users",
			Columns: []pgd.Column{
				{Name: "id", Type: "integer", Nullable: "false"},
				{Name: "email", Type: "text"},
			},
			PK: &pgd.PrimaryKey{Name: "pk_users", Columns: []pgd.ColRef{{Name: "id"}}},
		}},
	}}}
	r := diff.Diff(old, updated)
	require.NotEmpty(t, r.Changes, "fixture must produce diff")
	return r
}

func TestRenderDiff_SQL(t *testing.T) {
	out, err := renderDiff(diffFixture(t), "sql")
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(string(out)), "ALTER TABLE")
	assert.Contains(t, string(out), "email")
}

func TestRenderDiff_JSON(t *testing.T) {
	out, err := renderDiff(diffFixture(t), "json")
	require.NoError(t, err)

	var parsed []map[string]any
	require.NoError(t, json.Unmarshal(out, &parsed))
	assert.NotEmpty(t, parsed)
}

func TestRenderDiff_UnknownFormatFallsBackToSQL(t *testing.T) {
	out, err := renderDiff(diffFixture(t), "yaml")
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(string(out)), "ALTER TABLE")
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantPos []string
		wantOut string
		wantF   string
	}{
		{"flags first", []string{"-o", "out.sql", "in.pgd"}, []string{"in.pgd"}, "out.sql", "sql"},
		{"flags last", []string{"in.pgd", "-o", "out.sql"}, []string{"in.pgd"}, "out.sql", "sql"},
		{"flags between", []string{"old.pgd", "-f", "json", "new.pgd", "-o=out.sql"}, []string{"old.pgd", "new.pgd"}, "out.sql", "json"},
		{"double dash", []string{"-o", "out.sql", "--", "-weird.pgd", "-f"}, []string{"-weird.pgd", "-f"}, "out.sql", "sql"},
		{"no args", nil, nil, "", "sql"},
		{"dsn", []string{"postgres://u@h/db", "-o", "db.pgd"}, []string{"postgres://u@h/db"}, "db.pgd", "sql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			out := fs.String("o", "", "")
			f := fs.String("f", "sql", "")
			pos := parseArgs(fs, tt.args)
			assert.Equal(t, tt.wantPos, pos)
			assert.Equal(t, tt.wantOut, *out)
			assert.Equal(t, tt.wantF, *f)
		})
	}
}

func TestConvertOutput(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("taken.pgd", nil, 0o644))

	tests := []struct {
		input, output string
		want          string
		wantErr       bool
	}{
		{"schema.sql", "", "schema.pgd", false},
		{"dir/Schema.SQL", "", "dir/Schema.pgd", false},
		{"schema.sql", "out.pgd", "out.pgd", false},
		{"postgres://u:p@h/mydb", "", "mydb.pgd", false},
		{"postgres://u:p@h/", "", "schema.pgd", false},
		{"postgres://u:p@h/a%2F..%2F..%2Fescaped", "", "escaped.pgd", false}, // stays in the current directory
		{"postgres://u:p@h/taken", "", "", true},                             // never overwrites a model
		{"postgres://u:p@h/taken", "taken.pgd", "taken.pgd", false},
	}
	for _, tt := range tests {
		t.Run(tt.input+" "+tt.output, func(t *testing.T) {
			got, err := convertOutput(tt.input, tt.output)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

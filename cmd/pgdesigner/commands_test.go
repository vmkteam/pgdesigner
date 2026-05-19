package main

import (
	"encoding/json"
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

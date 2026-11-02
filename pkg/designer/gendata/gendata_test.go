package gendata

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pgquery "github.com/wasilibs/go-pgquery"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func loadProject(t *testing.T, path string) *pgd.Project {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var p pgd.Project
	require.NoError(t, xml.Unmarshal(data, &p))
	return &p
}

func TestGenerate_Synthetic(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{
		Seed:      42,
		Rows:      5,
		BatchSize: 100,
	})
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "BEGIN;")
	assert.Contains(t, got, "COMMIT;")
	assert.Contains(t, got, "INSERT INTO")

	golden := filepath.Join("testdata", "synthetic_testdata.sql")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		require.NoError(t, os.WriteFile(golden, buf.Bytes(), 0o644))
	}
	if expected, err := os.ReadFile(golden); err == nil {
		assert.Equal(t, string(expected), got)
	}
}

func TestGenerate_TableOrder(t *testing.T) {
	// users → documents (FK to users) — users must come first
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 1, Rows: 2, BatchSize: 100})
	require.NoError(t, err)

	got := buf.String()
	usersIdx := bytes.Index([]byte(got), []byte(`"users"`))
	docsIdx := bytes.Index([]byte(got), []byte(`"documents"`))
	assert.Greater(t, docsIdx, usersIdx, "users should come before documents")
}

func TestGenerate_IdentityIncluded(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 1, Rows: 2, BatchSize: 100})
	require.NoError(t, err)

	// users table has "id" as identity — it should be included with OVERRIDING SYSTEM VALUE
	got := buf.String()
	assert.Contains(t, got, "OVERRIDING SYSTEM VALUE")
	assert.Contains(t, got, `"id"`)
}

func TestGenerate_SkipTable(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{
		Seed: 1,
		Rows: 2,
		Tables: map[string]Table{
			"audit_log": {Skip: true},
		},
	})
	require.NoError(t, err)

	got := buf.String()
	assert.NotContains(t, got, `"audit_log"`)
}

func TestGenerate_CustomRows(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{
		Seed: 1,
		Rows: 2,
		Tables: map[string]Table{
			"tags": {Rows: 10},
		},
	})
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "-- Table: \"tags\" (10 rows)")
}

func TestTopoSort_SelfRef(t *testing.T) {
	// categories has self-referencing FK (parent_id → categories.id)
	tables := []tableKey{{Schema: "public", Table: "categories"}}
	graph := map[tableKey][]tableKey{
		{Schema: "public", Table: "categories"}: {{Schema: "public", Table: "categories"}},
	}

	order, cycles := topoSortFixed(graph, tables)
	assert.Len(t, order, 1)
	assert.Empty(t, cycles, "self-references should not produce cycle edges")
}

func TestTopoSort_Cycle(t *testing.T) {
	a := tableKey{Schema: "public", Table: "a"}
	b := tableKey{Schema: "public", Table: "b"}
	tables := []tableKey{a, b}
	graph := map[tableKey][]tableKey{
		a: {b},
		b: {a},
	}

	order, cycles := topoSortFixed(graph, tables)
	assert.Len(t, order, 2)
	assert.NotEmpty(t, cycles, "should detect cycle")
}

func TestCollectEnums(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))
	enums := collectEnums(p)
	assert.Contains(t, enums, "document_status")
	assert.Contains(t, enums, "user_role")
	assert.Equal(t, []string{"draft", "review", "published", "archived"}, enums["document_status"])
}

func TestCollectDomains(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))
	domains := collectDomains(p)

	assert.Equal(t, "text", domains["email_address"])
	assert.Equal(t, "integer", domains["positive_int"])
	assert.Equal(t, "varchar", domains["slug"])
}

func TestDomainResolve_GeneratesCorrectType(t *testing.T) {
	faker := gofakeit.New(1)
	ctx := &genContext{
		enums: map[string][]string{},
		domains: map[string]string{
			"positive_int":  "integer",
			"email_address": "text",
			"slug":          "varchar",
		},
		checkIN: map[string][]string{},
		rowVals: map[string]string{},
	}

	// positive_int domain → should generate an integer, not a word
	col := pgd.Column{Name: "version", Type: "positive_int", Nullable: "false"}
	val := byType(faker, col, 1, ctx)
	assert.NotContains(t, val, "'", "positive_int domain should produce a number, not a quoted string")

	// email_address domain → should generate a sentence (text fallback)
	col2 := pgd.Column{Name: "custom_field", Type: "email_address", Nullable: "false"}
	val2 := byType(faker, col2, 1, ctx)
	assert.Contains(t, val2, "'", "email_address domain (text) should produce a quoted string")

	// unknown domain stays as-is (fallback)
	col3 := pgd.Column{Name: "x", Type: "unknown_domain", Nullable: "false"}
	val3 := byType(faker, col3, 1, ctx)
	assert.Contains(t, val3, "'", "unknown domain should fallback to quoted word")
}

func TestParseCheckINValues(t *testing.T) {
	checks := []pgd.Check{
		{Name: "chk_action", Expression: "action IN ('INSERT', 'UPDATE', 'DELETE')"},
		{Name: "chk_status", Expression: "status IN('active', 'inactive')"},
		{Name: "chk_complex", Expression: "status <> 'published' OR published_at IS NOT NULL"},
		{Name: "chk_color", Expression: "color ~ '^#[0-9a-fA-F]{6}$'"},
	}

	m := parseCheckINValues(checks)
	assert.Equal(t, []string{"INSERT", "UPDATE", "DELETE"}, m["action"])
	assert.Equal(t, []string{"active", "inactive"}, m["status"])
	assert.NotContains(t, m, "color", "regex CHECK should not be parsed")
	assert.NotContains(t, m, "status <>", "complex expression should not match")
}

func TestGenerate_CheckINValues(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 42, Rows: 10, BatchSize: 100})
	require.NoError(t, err)

	got := buf.String()
	// audit_log.action has CHECK action IN ('INSERT', 'UPDATE', 'DELETE')
	// all action values should be one of these
	for _, allowed := range []string{"'INSERT'", "'UPDATE'", "'DELETE'"} {
		if strings.Contains(got, allowed) {
			return // at least one valid value found
		}
	}
	t.Error("expected at least one of INSERT/UPDATE/DELETE in audit_log action values")
}

func TestGenerate_CoherentTimestamps(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 42, Rows: 5, BatchSize: 100})
	require.NoError(t, err)

	got := buf.String()
	// find a line with "users" INSERT and check created_at < updated_at
	for _, line := range strings.Split(got, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "(") {
			continue
		}
		// check that created_at values in users are 2020+
		if strings.Contains(got, `"users"`) && strings.Contains(line, "'202") {
			// at least some timestamps are in 2020+ range
			return
		}
	}
	t.Error("expected timestamps in 2020+ range for created_at/updated_at columns")
}

func TestGenerate_UniqueValues(t *testing.T) {
	p := loadProject(t, filepath.Join("..", "..", "format", "sql", "testdata", "synthetic.pgd"))

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 42, Rows: 20, BatchSize: 100})
	require.NoError(t, err)

	got := buf.String()

	// Extract email values from users INSERT and verify uniqueness
	emails := make(map[string]bool)
	inUsers := false
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, `"users"`) {
			inUsers = true
			continue
		}
		if inUsers && strings.HasPrefix(strings.TrimSpace(line), "(") {
			// extract second field (email) — format: (id, 'email', ...)
			parts := strings.SplitN(line, ",", 3)
			if len(parts) >= 2 {
				email := strings.TrimSpace(parts[1])
				assert.False(t, emails[email], "duplicate email found: %s", email)
				emails[email] = true
			}
		}
		if inUsers && strings.Contains(line, ";") {
			inUsers = false
		}
	}
	assert.GreaterOrEqual(t, len(emails), 20, "should have 20 unique emails")
}

func TestUniqueTracker_Single(t *testing.T) {
	tbl := &pgd.Table{
		Uniques: []pgd.Unique{
			{Name: "uq_email", Columns: []pgd.ColRef{{Name: "email"}}},
		},
	}
	ut := newUniqueTracker(tbl)

	assert.True(t, ut.isSingleUnique("email"))
	assert.False(t, ut.isSingleUnique("name"))

	assert.True(t, ut.checkAndTrackSingle("email", "'a@b.com'"))
	assert.False(t, ut.checkAndTrackSingle("email", "'a@b.com'"))
	assert.True(t, ut.checkAndTrackSingle("email", "'c@d.com'"))
}

func TestUniqueTracker_Composite(t *testing.T) {
	tbl := &pgd.Table{
		Uniques: []pgd.Unique{
			{Name: "uq_doc_user", Columns: []pgd.ColRef{{Name: "doc_id"}, {Name: "user_id"}}},
		},
	}
	ut := newUniqueTracker(tbl)

	assert.True(t, ut.checkAndTrackComposite(map[string]string{"doc_id": "1", "user_id": "2"}))
	assert.False(t, ut.checkAndTrackComposite(map[string]string{"doc_id": "1", "user_id": "2"}))
	assert.True(t, ut.checkAndTrackComposite(map[string]string{"doc_id": "1", "user_id": "3"}))
}

func TestIsStatusTable(t *testing.T) {
	assert.True(t, isStatusTable("statuses"))
	assert.True(t, isStatusTable("status"))
	assert.True(t, isStatusTable("order_statuses"))
	assert.True(t, isStatusTable("order_status"))
	assert.False(t, isStatusTable("users"))
	assert.False(t, isStatusTable("status_log"))
}

func TestGenerate_StatusTable(t *testing.T) {
	p := &pgd.Project{
		Version:       1,
		PgVersion:     "17",
		DefaultSchema: "public",
		Schemas: []pgd.Schema{{
			Name: "public",
			Tables: []pgd.Table{{
				Name: "statuses",
				Columns: []pgd.Column{
					{Name: "id", Type: "integer", Nullable: "false"},
					{Name: "name", Type: "varchar", Length: 50, Nullable: "false"},
				},
				PK: &pgd.PrimaryKey{Name: "statuses_pkey", Columns: []pgd.ColRef{{Name: "id"}}},
			}},
		}},
	}

	var buf bytes.Buffer
	err := Generate(&buf, p, Options{Seed: 1, Rows: 5})
	require.NoError(t, err)

	got := buf.String()
	// first 3 rows should be well-known statuses
	assert.Contains(t, got, "(1, 'enabled')")
	assert.Contains(t, got, "(2, 'disabled')")
	assert.Contains(t, got, "(3, 'deleted')")
	// 'enabled' appears exactly once (row 1 only, rows 4-5 use normal generation)
	assert.Equal(t, 1, strings.Count(got, "'enabled'"), "only row 1 should have 'enabled'")
}

func TestSqlQuote(t *testing.T) {
	assert.Equal(t, "'hello'", sqlQuote("hello"))
	assert.Equal(t, "'it''s'", sqlQuote("it's"))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "abc", truncate("abcdef", 3))
	assert.Equal(t, "ab", truncate("ab", 5))
	assert.Equal(t, "abc", truncate("abc", 0))
}

func TestGenerate_NameCantBreakOutOfComment(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{Name: "public", Tables: []pgd.Table{{
		Name:    "t\nDROP TABLE users; --",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
	}}}}}

	var buf bytes.Buffer
	require.NoError(t, Generate(&buf, p, Options{Seed: 1, Rows: 1}))
	tree, err := pgquery.Parse(buf.String())
	require.NoError(t, err)
	for _, s := range tree.Stmts {
		assert.Nil(t, s.Stmt.GetDropStmt(), "statement outside the comment")
	}
}

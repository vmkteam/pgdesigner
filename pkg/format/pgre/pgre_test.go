package pgre

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	sqlconv "github.com/vmkteam/pgdesigner/pkg/format/sql"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

const testDSNEnv = "PGD_TEST_DSN"

// testDSN returns base DSN for test PostgreSQL server.
// Defaults to postgres://postgres@localhost:5432/?sslmode=disable
func testDSN() string {
	if v := os.Getenv(testDSNEnv); v != "" {
		return v
	}
	return "postgres://postgres@localhost:5432/?sslmode=disable"
}

func dbDSN(base, dbName string) string {
	// replace path in DSN
	idx := strings.LastIndex(base, "/")
	if idx < 0 {
		return base + "/" + dbName
	}
	// find query part
	qIdx := strings.Index(base[idx:], "?")
	if qIdx < 0 {
		return base[:idx+1] + dbName
	}
	return base[:idx+1] + dbName + base[idx+qIdx:]
}

func createDB(t *testing.T, name string) string {
	t.Helper()
	dsn := testDSN()
	// use psql to create/drop DB
	run(t, "psql", dsn, "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	run(t, "psql", dsn, "-c", fmt.Sprintf("CREATE DATABASE %s", name))
	return dbDSN(dsn, name)
}

func dropDB(t *testing.T, name string) {
	t.Helper()
	run(t, "psql", testDSN(), "-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
}

func loadSQL(t *testing.T, dsn, sqlFile string) {
	t.Helper()
	run(t, "psql", dsn, "-f", sqlFile)
}

func run(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s %v failed: %v", name, args[:2], err)
	}
}

// skipIfNoPG skips the test if PostgreSQL is not reachable.
func skipIfNoPG(t *testing.T) {
	t.Helper()
	cmd := exec.Command("psql", testDSN(), "-c", "SELECT 1")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("PostgreSQL not reachable at %s: %v", testDSN(), err)
	}
}

func TestIsDSN(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"postgres://u@h/db", true},
		{"postgresql://u@h/db", true},
		{"postgres://", true},
		{"abcdefgh.pgd", false}, // 12 chars used to panic
		{"abcdefg.pgd", false},
		{"abcdefghi.pgd", false},
		{"postgres.pgd", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, IsDSN(tt.in))
		})
	}
}

func TestRedactDSN(t *testing.T) {
	assert.Equal(t, "postgres://u:xxxxx@h:5432/db", RedactDSN("postgres://u:S3cret@h:5432/db"))
	assert.Equal(t, "file.pgd", RedactDSN("file.pgd"))

	_, err := parseDSN("postgres://u:S3cret@h:badport/db")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "S3cret")
}

func TestDatabaseName(t *testing.T) {
	assert.Equal(t, "mydb", DatabaseName("postgres://u:p@localhost:5432/mydb?sslmode=disable"))
	assert.Empty(t, DatabaseName("postgres://u@localhost:5432/"))
	assert.Empty(t, DatabaseName("postgres://u@localhost"))
}

func TestParseIndexKeys(t *testing.T) {
	tests := []struct {
		name string
		def  string
		want []pgd.IndexKey
	}{
		{
			name: "plain column",
			def:  `CREATE INDEX foo ON t ("name")`,
			want: []pgd.IndexKey{{Name: "name"}},
		},
		{
			name: "desc nulls last",
			def:  `CREATE INDEX foo ON t ("name" DESC NULLS LAST)`,
			want: []pgd.IndexKey{{Name: "name", Order: "desc", Nulls: "last"}},
		},
		{
			name: "gin with opclass",
			def:  `CREATE INDEX foo ON t USING gin ("query" gin_trgm_ops)`,
			want: []pgd.IndexKey{{Name: "query", Opclass: "gin_trgm_ops"}},
		},
		{
			name: "btree opclass with desc",
			def:  `CREATE INDEX foo ON t ("email" text_pattern_ops DESC)`,
			want: []pgd.IndexKey{{Name: "email", Order: "desc", Opclass: "text_pattern_ops"}},
		},
		{
			name: "multi-column with mixed opclass",
			def:  `CREATE INDEX foo ON t ("a" int4_ops, "b" DESC)`,
			want: []pgd.IndexKey{{Name: "a", Opclass: "int4_ops"}, {Name: "b", Order: "desc"}},
		},
		{
			name: "expression index",
			def:  `CREATE INDEX foo ON t ((lower("name")))`,
			want: []pgd.IndexKey{{Expression: `(lower(name))`}},
		},
		{
			name: "columns and expressions interleaved",
			def:  `CREATE INDEX foo ON t USING btree ("shopId", COALESCE("updatedAt", "createdAt") DESC, "statusId")`,
			want: []pgd.IndexKey{{Name: "shopId"}, {Expression: `(COALESCE("updatedAt", "createdAt"))`, Order: "desc"}, {Name: "statusId"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIndexKeys(tt.def)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertColumn_Generated(t *testing.T) {
	tests := []struct {
		attGenerated string
		want         *pgd.Generated
	}{
		{"s", &pgd.Generated{Expression: "(first_name || ' '::text) || last_name", Stored: "true"}},
		{"v", &pgd.Generated{Expression: "(first_name || ' '::text) || last_name", Stored: "false"}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.attGenerated, func(t *testing.T) {
			// pg_get_expr output, normalized like the SQL converter stores it
			col := convertColumn(pgColumn{Name: "full_name", Type: "text", AttGenerated: tt.attGenerated,
				DefaultValue: "((first_name || ' '::text) || last_name)"})
			assert.Equal(t, tt.want, col.Generated)
		})
	}
}

func TestParseRelOptions(t *testing.T) {
	assert.Equal(t, []pgd.WithParam{{Name: "fillfactor", Value: "80"}, {Name: "toast.autovacuum_enabled", Value: "false"}},
		parseRelOptions([]string{"fillfactor=80", "toast.autovacuum_enabled=false"}))
	assert.Empty(t, parseRelOptions(nil))
}

func TestConvertIndex_ExpressionOptions(t *testing.T) {
	idx := convertIndex("t", pgIndex{
		IndexName: "ix_t_mnn_trgm",
		Method:    "gin",
		Def:       `CREATE INDEX ix_t_mnn_trgm ON public.t USING gin (((params ->> 'mnn'::text)) gin_trgm_ops)`,
	})
	assert.Equal(t, []pgd.IndexKey{{Expression: "(params ->> 'mnn'::text)", Opclass: "gin_trgm_ops"}}, idx.Keys)

	idx = convertIndex("t", pgIndex{
		IndexName: "ix_t_lower_desc",
		Method:    "btree",
		Def:       `CREATE INDEX ix_t_lower_desc ON public.t USING btree (lower(title) DESC NULLS LAST)`,
	})
	assert.Equal(t, []pgd.IndexKey{{Expression: "(lower(title))", Order: "desc", Nulls: "last"}}, idx.Keys)
}

func TestParseIndexExtras(t *testing.T) {
	tests := []struct {
		name             string
		def              string
		wantInclude      []pgd.ColRef
		wantNullsNotDist bool
		wantTablespace   string
	}{
		{
			name: "plain index",
			def:  `CREATE INDEX foo ON t ("name")`,
		},
		{
			name:        "include columns",
			def:         `CREATE UNIQUE INDEX foo ON t ("customerId") INCLUDE ("totalAmount", "createdAt")`,
			wantInclude: []pgd.ColRef{{Name: "totalAmount"}, {Name: "createdAt"}},
		},
		{
			name:             "nulls not distinct",
			def:              `CREATE UNIQUE INDEX foo ON t ("statusId") NULLS NOT DISTINCT`,
			wantNullsNotDist: true,
		},
		{
			name:           "tablespace",
			def:            `CREATE INDEX foo ON t ("statusId") TABLESPACE fastssd`,
			wantTablespace: "fastssd",
		},
		{
			name:             "all combined",
			def:              `CREATE UNIQUE INDEX foo ON t ("customerId") INCLUDE ("totalAmount") NULLS NOT DISTINCT WITH (fillfactor='80') TABLESPACE fastssd WHERE "statusId" = 1`,
			wantInclude:      []pgd.ColRef{{Name: "totalAmount"}},
			wantNullsNotDist: true,
			wantTablespace:   "fastssd",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInc, gotNND, gotTS := parseIndexExtras(tt.def)
			assert.Equal(t, tt.wantInclude, gotInc)
			assert.Equal(t, tt.wantNullsNotDist, gotNND)
			assert.Equal(t, tt.wantTablespace, gotTS)
		})
	}
}

func TestIntrospect_RoundTrip(t *testing.T) {
	skipIfNoPG(t)

	tests := []struct {
		name     string
		sqlFile  string // generated DDL to load
		origSQL  string // original SQL for sql.ParseSQL comparison
		project  string
		schemas  []string
		full     bool
		maxDiffs int // max allowed diffs (0 = must be zero)
	}{
		{
			name:    "chinook",
			sqlFile: "../../format/sql/testdata/chinook_generated.sql",
			origSQL: "../../format/sql/testdata/chinook.sql",
			project: "chinook",
			schemas: []string{"public"},
		},
		{
			name:    "northwind",
			sqlFile: "../../format/sql/testdata/northwind_generated.sql",
			origSQL: "../../format/sql/testdata/northwind.sql",
			project: "northwind",
			schemas: []string{"public"},
		},
		{
			name:     "airlines",
			sqlFile:  "../../format/sql/testdata/airlines_generated.sql",
			origSQL:  "../../format/sql/testdata/airlines.sql",
			project:  "airlines",
			schemas:  []string{"bookings"},
			full:     true,
			maxDiffs: 0,
		},
		{
			name:    "pagila",
			sqlFile: "../../format/sql/testdata/pagila_generated.sql",
			origSQL: "../../format/sql/testdata/pagila.sql",
			project: "pagila",
			schemas: []string{"public"},
			full:    true,
		},
		{
			name:    "adventureworks",
			sqlFile: "../../format/sql/testdata/adventureworks_generated.sql",
			origSQL: "../../format/sql/testdata/adventureworks.sql",
			project: "adventureworks",
			full:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbName := "pgd_test_" + tt.name

			// Create DB and load DDL
			dsn := createDB(t, dbName)
			defer dropDB(t, dbName)

			loadSQL(t, dsn, tt.sqlFile)

			// Introspect via pgre
			reProject, err := Connect(dsn, Options{
				Schemas: tt.schemas,
				Full:    tt.full,
			})
			require.NoError(t, err)

			// Parse original SQL
			sqlData, err := os.ReadFile(tt.origSQL)
			require.NoError(t, err)
			sqlProject, err := sqlconv.ParseSQL(string(sqlData), tt.project)
			require.NoError(t, err)

			// Normalize both for comparison
			normalizeProject(reProject)
			normalizeProject(sqlProject)

			// Compare table counts
			reTables := countTables(reProject)
			sqlTables := countTables(sqlProject)
			assert.Equal(t, sqlTables, reTables, "table count mismatch")

			// Compare FK counts
			reFKs := countFKs(reProject)
			sqlFKs := countFKs(sqlProject)
			assert.Equal(t, sqlFKs, reFKs, "FK count mismatch")

			// Diff: should be zero or within maxDiffs
			result := diff.Diff(sqlProject, reProject)
			diffSQL := result.SQL()
			if diffSQL != "" {
				lines := strings.Split(strings.TrimSpace(diffSQL), "\n\n")
				nDiffs := len(lines)
				t.Logf("diff has %d changes", nDiffs)
				if nDiffs > tt.maxDiffs {
					t.Errorf("too many diffs: got %d, max %d\n%s", nDiffs, tt.maxDiffs, diffSQL)
				}
			}
		})
	}
}

func normalizeProject(p *pgd.Project) {
	// Sort schemas by name
	sort.Slice(p.Schemas, func(i, j int) bool {
		return p.Schemas[i].Name < p.Schemas[j].Name
	})
	for si := range p.Schemas {
		// Sort tables by name
		sort.Slice(p.Schemas[si].Tables, func(i, j int) bool {
			return p.Schemas[si].Tables[i].Name < p.Schemas[si].Tables[j].Name
		})
		// Sort indexes by name
		sort.Slice(p.Schemas[si].Indexes, func(i, j int) bool {
			return p.Schemas[si].Indexes[i].Name < p.Schemas[si].Indexes[j].Name
		})
		for ti := range p.Schemas[si].Tables {
			p.Schemas[si].Tables[ti].Comment = ""
			for ci := range p.Schemas[si].Tables[ti].Columns {
				c := &p.Schemas[si].Tables[ti].Columns[ci]
				c.Storage = "" // pgre always returns it, SQL parser doesn't
				c.Comment = "" // pgre stores inline, SQL parser stores in p.Comments
				// Normalize serial: pgre sees integer+nextval, SQL parser sees serial
				if strings.HasPrefix(c.Default, "nextval(") {
					c.Default = ""
					switch c.Type {
					case "integer":
						c.Type = "serial"
					case "bigint":
						c.Type = "bigserial"
					case "smallint":
						c.Type = "smallserial"
					}
				}
				// Normalize defaults for fair comparison
				c.Default = strings.ReplaceAll(c.Default, "public.", "")
				// Strip type cast: 'value'::type → 'value'
				if idx := strings.Index(c.Default, "::"); idx > 0 && strings.HasPrefix(c.Default, "'") {
					c.Default = c.Default[:idx]
				}
				// Normalize now()/date variants (case-insensitive)
				switch strings.ToLower(c.Default) {
				case "('now'::text)::date", "current_date":
					c.Default = "CURRENT_DATE"
				case "now()", "current_timestamp":
					c.Default = "now()"
				}
			}
		}
	}
	// Clear metadata that differs
	p.ProjectMeta = pgd.ProjectMeta{}
	p.Layouts = pgd.Layouts{}
	p.PgVersion = ""
	p.Sequences = nil
	p.Comments = nil
	// Clear functions/triggers/views (pgre full mode may differ in representation)
	p.Functions = nil
	p.Triggers = nil
	if p.Views != nil {
		p.Views = nil
	}
	if p.Types != nil {
		p.Types = nil
	}
}

func countTables(p *pgd.Project) int {
	var n int
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			if t.PartitionOf == "" {
				n++
			}
		}
	}
	return n
}

func countFKs(p *pgd.Project) int {
	var n int
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			n += len(t.FKs)
		}
	}
	return n
}

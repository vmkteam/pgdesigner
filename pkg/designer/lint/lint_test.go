package lint

import (
	"encoding/xml"
	"fmt"
	"os"
	"testing"

	"github.com/vmkteam/pgdesigner/pkg/pgd"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func hasCode(issues []Issue, code string) bool {
	for _, i := range issues {
		if i.Code == code {
			return true
		}
	}
	return false
}

func countCode(issues []Issue, code string) int {
	n := 0
	for _, i := range issues {
		if i.Code == code {
			n++
		}
	}
	return n
}

func countSeverity(issues []Issue, s Severity) int {
	n := 0
	for _, i := range issues {
		if i.Severity == s {
			n++
		}
	}
	return n
}

// minProject returns a valid project with one table and PK — zero issues expected.
func minProject() *pgd.Project {
	return &pgd.Project{
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
		}},
	}
}

// --- clean project ---

func TestValidate_Clean(t *testing.T) {
	issues := Validate(minProject())
	assert.Equal(t, 0, countSeverity(issues, Error), "unexpected errors: %v", issues)
}

// --- E rules ---

func TestValidate_E001_EmptyName(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "", Type: "text"})
	assert.True(t, hasCode(Validate(p), RuleEmptyName))
}

func TestValidate_E002_LongIdentifier(t *testing.T) {
	p := minProject()
	long := "a123456789012345678901234567890123456789012345678901234567890abcd" // 64 chars
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: long, Type: "text"})
	assert.True(t, hasCode(Validate(p), RuleIdentTooLong))
}

func TestValidate_E003_DuplicateTable(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables = append(p.Schemas[0].Tables, pgd.Table{
		Name:    "users",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
	})
	assert.True(t, hasCode(Validate(p), RuleDupTable))
}

func TestValidate_E004_DuplicateColumn(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "id", Type: "text"})
	assert.True(t, hasCode(Validate(p), RuleDupColumn))
}

func TestValidate_E005_DuplicateIndexName(t *testing.T) {
	p := minProject()
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_a", Table: "users", Columns: []pgd.ColRef{{Name: "id"}}},
		{Name: "idx_a", Table: "users", Columns: []pgd.ColRef{{Name: "name"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleDupIndexName))
}

func TestValidate_E006_DuplicateConstraint(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Uniques = []pgd.Unique{
		{Name: "pk_users", Columns: []pgd.ColRef{{Name: "name"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleDupConstraint))
}

func TestValidate_E007_PKColumnNotFound(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].PK.Columns = []pgd.ColRef{{Name: "nonexistent"}}
	assert.True(t, hasCode(Validate(p), RulePKColNotFound))
}

func TestValidate_E008_E009_E010_FK(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables = append(p.Schemas[0].Tables, pgd.Table{
		Name:    "posts",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "user_id", Type: "integer"}},
		PK:      &pgd.PrimaryKey{Name: "pk_posts", Columns: []pgd.ColRef{{Name: "id"}}},
		FKs: []pgd.ForeignKey{{
			Name: "fk_bad_local", ToTable: "users",
			Columns: []pgd.FKCol{{Name: "bad_col", References: "id"}},
		}},
	})
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleFKColNotFound), "expected E008 for bad local column")

	// E009: unknown target table
	p.Schemas[0].Tables[1].FKs[0] = pgd.ForeignKey{
		Name: "fk_bad_table", ToTable: "nonexistent",
		Columns: []pgd.FKCol{{Name: "user_id", References: "id"}},
	}
	issues = Validate(p)
	assert.True(t, hasCode(issues, RuleFKTableNotFound), "expected E009 for unknown table")

	// E010: unknown target column
	p.Schemas[0].Tables[1].FKs[0] = pgd.ForeignKey{
		Name: "fk_bad_ref", ToTable: "users",
		Columns: []pgd.FKCol{{Name: "user_id", References: "nonexistent"}},
	}
	issues = Validate(p)
	assert.True(t, hasCode(issues, RuleFKRefColNotFound), "expected E010 for unknown ref column")
}

func TestValidate_E011_E012_IndexColumn(t *testing.T) {
	p := minProject()
	// E012: unknown table
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_bad", Table: "nonexistent", Columns: []pgd.ColRef{{Name: "id"}}},
	}
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleIdxTableNotFound), "expected E012")

	// E011: unknown column
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_bad", Table: "users", Columns: []pgd.ColRef{{Name: "nonexistent"}}},
	}
	issues = Validate(p)
	assert.True(t, hasCode(issues, RuleIdxColNotFound), "expected E011")
}

func TestValidate_E013_UniqueColumnNotFound(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Uniques = []pgd.Unique{
		{Name: "uq_bad", Columns: []pgd.ColRef{{Name: "nonexistent"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleUniqueColNotFound))
}

func TestValidate_E014_E015_Enum(t *testing.T) {
	p := minProject()
	p.Types = &pgd.Types{Enums: []pgd.Enum{{Name: "empty_enum"}}}
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleEmptyEnum), "expected E014 for empty enum")

	p.Types.Enums = []pgd.Enum{{Name: "dup_enum", Labels: []string{"a", "b", "a"}}}
	issues = Validate(p)
	assert.True(t, hasCode(issues, RuleDupEnumLabel), "expected E015 for duplicate label")
}

func TestValidate_E016_EmptyComposite(t *testing.T) {
	p := minProject()
	p.Types = &pgd.Types{Composites: []pgd.Composite{{Name: "empty_comp"}}}
	assert.True(t, hasCode(Validate(p), RuleEmptyComposite))
}

func TestValidate_E017_TableNoColumns(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{Name: "public", Tables: []pgd.Table{{Name: "empty"}}}}}
	assert.True(t, hasCode(Validate(p), RuleTableNoCols))
}

func TestValidate_E018_UnknownType(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "x", Type: "foobar"})
	assert.True(t, hasCode(Validate(p), RuleUnknownType))
}

func TestValidate_E018_UserTypeNotUnknown(t *testing.T) {
	p := minProject()
	p.Types = &pgd.Types{Enums: []pgd.Enum{{Name: "status", Labels: []string{"a", "b"}}}}
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "s", Type: "status"})
	assert.False(t, hasCode(Validate(p), RuleUnknownType), "user type should be known")
}

func TestValidate_E019_IncludeColumnNotFound(t *testing.T) {
	p := minProject()
	p.Schemas[0].Indexes = []pgd.Index{{
		Name: "idx_inc", Table: "users",
		Columns: []pgd.ColRef{{Name: "id"}},
		Include: &pgd.Include{Columns: []pgd.ColRef{{Name: "nonexistent"}}},
	}}
	assert.True(t, hasCode(Validate(p), RuleIncludeColNotFound))
}

func TestValidate_E020_DuplicateSchema(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{
		{Name: "public", Tables: []pgd.Table{{Name: "t", Columns: []pgd.Column{{Name: "id", Type: "integer"}}, PK: &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}}}}},
		{Name: "public"},
	}}
	assert.True(t, hasCode(Validate(p), RuleDupSchema))
}

func TestValidate_E021_FKColumnCountZero(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables = append(p.Schemas[0].Tables, pgd.Table{
		Name:    "posts",
		Columns: []pgd.Column{{Name: "id", Type: "integer"}},
		PK:      &pgd.PrimaryKey{Name: "pk_posts", Columns: []pgd.ColRef{{Name: "id"}}},
		FKs:     []pgd.ForeignKey{{Name: "fk_empty", ToTable: "users"}},
	})
	assert.True(t, hasCode(Validate(p), RuleFKNoCols))
}

func TestValidate_E025_SequenceType(t *testing.T) {
	p := minProject()
	p.Sequences = []pgd.Sequence{{Name: "seq_bad", Type: "varchar", OwnedBy: "users.id"}}
	assert.True(t, hasCode(Validate(p), RuleSeqTypeInvalid))
}

func TestValidate_E030_ViewNoQuery(t *testing.T) {
	p := minProject()
	p.Views = &pgd.Views{Views: []pgd.View{{Name: "v_empty"}}}
	assert.True(t, hasCode(Validate(p), RuleViewNoQuery))
}

// --- W rules ---

func TestValidate_W001_FKTypeMismatch(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "parents",
				Columns: []pgd.Column{{Name: "id", Type: "bigint"}},
				PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			},
			{
				Name:    "children",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "parent_id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk2", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs: []pgd.ForeignKey{{
					Name: "fk_parent", ToTable: "parents",
					Columns: []pgd.FKCol{{Name: "parent_id", References: "id"}},
				}},
			},
		},
	}}}
	assert.True(t, hasCode(Validate(p), RuleFKTypeMismatch))
}

func TestValidate_W002_MissingFKIndex(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "parents",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			},
			{
				Name:    "children",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "parent_id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk2", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs: []pgd.ForeignKey{{
					Name: "fk_parent", ToTable: "parents",
					Columns: []pgd.FKCol{{Name: "parent_id", References: "id"}},
				}},
			},
		},
	}}}
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleMissingFKIndex), "expected W002 for missing FK index")

	// add index — W002 should disappear
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_parent", Table: "children", Columns: []pgd.ColRef{{Name: "parent_id"}}},
	}
	issues = Validate(p)
	assert.False(t, hasCode(issues, RuleMissingFKIndex), "W002 should not fire with index")
}

func TestValidate_W003_FKCycle(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "a",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "b_id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk_a", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs:     []pgd.ForeignKey{{Name: "fk_ab", ToTable: "b", Columns: []pgd.FKCol{{Name: "b_id", References: "id"}}}},
			},
			{
				Name:    "b",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "a_id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk_b", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs:     []pgd.ForeignKey{{Name: "fk_ba", ToTable: "a", Columns: []pgd.FKCol{{Name: "a_id", References: "id"}}}},
			},
		},
	}}}
	assert.True(t, hasCode(Validate(p), RuleCircularFK))
}

func TestValidate_W004_NoPK(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name:   "public",
		Tables: []pgd.Table{{Name: "t", Columns: []pgd.Column{{Name: "id", Type: "integer"}}}},
	}}}
	assert.True(t, hasCode(Validate(p), RuleNoPK))
}

func TestValidate_W005_DuplicateIndexColumns(t *testing.T) {
	p := minProject()
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_1", Table: "users", Columns: []pgd.ColRef{{Name: "name"}}},
		{Name: "idx_2", Table: "users", Columns: []pgd.ColRef{{Name: "name"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleDupIndexCols))
}

func TestValidate_W015_FKNoAction(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "parents",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			},
			{
				Name:    "children",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "pid", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk2", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs: []pgd.ForeignKey{{
					Name: "fk_no_action", ToTable: "parents",
					OnDelete: "no action", OnUpdate: "no action",
					Columns: []pgd.FKCol{{Name: "pid", References: "id"}},
				}},
			},
		},
	}}}
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleFKNoAction), "expected W015 for NO ACTION")

	// restrict should NOT trigger W015
	p.Schemas[0].Tables[1].FKs[0].OnDelete = "restrict"
	p.Schemas[0].Tables[1].FKs[0].OnUpdate = "restrict"
	issues = Validate(p)
	assert.False(t, hasCode(issues, RuleFKNoAction), "RESTRICT should not trigger W015")
}

func TestValidate_W012_NullablePK(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns[0].Nullable = "true"
	assert.True(t, hasCode(Validate(p), RulePKNullable))
}

// --- new W rules ---

func TestValidate_W016_FKToNonUnique(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "parents",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "name", Type: "text"}},
				PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			},
			{
				Name:    "children",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "pname", Type: "text"}},
				PK:      &pgd.PrimaryKey{Name: "pk2", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs: []pgd.ForeignKey{{
					Name: "fk_name", ToTable: "parents",
					Columns: []pgd.FKCol{{Name: "pname", References: "name"}},
				}},
			},
		},
	}}}
	assert.True(t, hasCode(Validate(p), RuleFKToNonUnique), "expected W016 for FK to non-unique column")
}

func TestValidate_W017_OverlappingIndexes(t *testing.T) {
	p := minProject()
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_short", Table: "users", Columns: []pgd.ColRef{{Name: "name"}}},
		{Name: "idx_long", Table: "users", Columns: []pgd.ColRef{{Name: "name"}, {Name: "id"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleOverlapIndex), "expected W017 for overlapping index")
}

func TestValidate_W018_DuplicateFK(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{
			{
				Name:    "parents",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			},
			{
				Name:    "children",
				Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "pid", Type: "integer"}},
				PK:      &pgd.PrimaryKey{Name: "pk2", Columns: []pgd.ColRef{{Name: "id"}}},
				FKs: []pgd.ForeignKey{
					{Name: "fk1", ToTable: "parents", Columns: []pgd.FKCol{{Name: "pid", References: "id"}}},
					{Name: "fk2", ToTable: "parents", Columns: []pgd.FKCol{{Name: "pid", References: "id"}}},
				},
			},
		},
	}}}
	assert.True(t, hasCode(Validate(p), RuleDupFK))
}

func TestValidate_W019_SelfFKNotNull(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{{
			Name:    "nodes",
			Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "parent_id", Type: "integer", Nullable: "false"}},
			PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
			FKs: []pgd.ForeignKey{{
				Name: "fk_self", ToTable: "nodes",
				Columns: []pgd.FKCol{{Name: "parent_id", References: "id"}},
			}},
		}},
	}}}
	assert.True(t, hasCode(Validate(p), RuleSelfFKNotNull))
}

func TestValidate_W020_ReservedWord(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{{
			Name:    "user",
			Columns: []pgd.Column{{Name: "id", Type: "integer"}, {Name: "order", Type: "integer"}},
			PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "id"}}},
		}},
	}}}
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleReservedWord), "expected W020 for reserved words")
	assert.Equal(t, 2, countCode(issues, RuleReservedWord), "both 'user' and 'order' should trigger W020")
}

func TestValidate_I009_JsonToJsonb(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "data", Type: "json"})
	assert.True(t, hasCode(Validate(p), RulePreferJsonb))
}

func TestValidate_I012_TextPK(t *testing.T) {
	p := &pgd.Project{Schemas: []pgd.Schema{{
		Name: "public",
		Tables: []pgd.Table{{
			Name:    "slugs",
			Columns: []pgd.Column{{Name: "slug", Type: "text"}},
			PK:      &pgd.PrimaryKey{Name: "pk", Columns: []pgd.ColRef{{Name: "slug"}}},
		}},
	}}}
	assert.True(t, hasCode(Validate(p), RuleTextPK))
}

func TestValidate_I013_TooManyIndexes(t *testing.T) {
	p := minProject()
	for i := range 12 {
		col := pgd.Column{Name: fmt.Sprintf("col%d", i), Type: "integer"}
		p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns, col)
		p.Schemas[0].Indexes = append(p.Schemas[0].Indexes, pgd.Index{
			Name: fmt.Sprintf("idx_%d", i), Table: "users",
			Columns: []pgd.ColRef{{Name: col.Name}},
		})
	}
	assert.True(t, hasCode(Validate(p), RuleTooManyIndexes))
}

func TestValidate_I016_IndexOnBoolean(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "active", Type: "boolean"})
	p.Schemas[0].Indexes = []pgd.Index{
		{Name: "idx_active", Table: "users", Columns: []pgd.ColRef{{Name: "active"}}},
	}
	assert.True(t, hasCode(Validate(p), RuleIndexOnBool))
}

func TestValidate_I001_CharN(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "code", Type: "char", Length: 3})
	assert.True(t, hasCode(Validate(p), RulePreferText))
}

func TestValidate_I004_Serial(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "old_id", Type: "serial"})
	assert.True(t, hasCode(Validate(p), RulePreferIdentity))
}

func TestValidate_I005_Timestamp(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "created", Type: "timestamp"})
	assert.True(t, hasCode(Validate(p), RulePreferTSTZ))
}

func TestValidate_I007_Rules(t *testing.T) {
	p := minProject()
	p.Rules = []pgd.Rule{{Name: "r1", Table: "users", Event: "select", Actions: "NOTHING"}}
	assert.True(t, hasCode(Validate(p), RuleAvoidRules))
}

// --- naming ---

func TestValidate_W007_Naming(t *testing.T) {
	p := minProject()
	p.ProjectMeta.Settings.Naming.Convention = "snake_case"
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "firstName", Type: "text"})
	issues := Validate(p)
	assert.True(t, hasCode(issues, RuleNamingViolation), "expected W007 for snake_case violation")
}

// --- ignore rules ---

func TestValidate_IgnoreRules(t *testing.T) {
	p := minProject()
	p.Schemas[0].Tables[0].Columns = append(p.Schemas[0].Tables[0].Columns,
		pgd.Column{Name: "data", Type: "json"})
	issues := Validate(p)
	assert.True(t, hasCode(issues, RulePreferJsonb), "I009 should fire without ignore")

	p.ProjectMeta.Settings.Lint = &pgd.Lint{IgnoreRules: "I009,W004"}
	issues = Validate(p)
	assert.False(t, hasCode(issues, RulePreferJsonb), "I009 should be suppressed")
}

// --- integration: existing .pgd files ---

func TestValidate_ExistingFiles(t *testing.T) {
	files := []struct {
		name string
		file string
	}{
		{"chinook", "../../format/sql/testdata/chinook.pgd"},
		{"northwind", "../../format/sql/testdata/northwind.pgd"},
		{"pagila", "../../format/sql/testdata/pagila.pgd"},
		{"airlines", "../../format/sql/testdata/airlines.pgd"},
		{"adventureworks", "../../format/sql/testdata/adventureworks.pgd"},
	}

	for _, tt := range files {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.file)
			require.NoError(t, err)

			var p pgd.Project
			require.NoError(t, xml.Unmarshal(data, &p))

			issues := Validate(&p)
			errors := countSeverity(issues, Error)
			warnings := countSeverity(issues, Warning)
			infos := countSeverity(issues, Info)

			t.Logf("%s: %d errors, %d warnings, %d info", tt.name, errors, warnings, infos)
			for _, i := range issues {
				if i.Severity == Error {
					t.Logf("  %s", i)
				}
			}
		})
	}
}

// --- known types ---

func TestIsKnownBuiltinType(t *testing.T) {
	known := []string{
		"integer", "bigint", "text", "boolean", "uuid", "jsonb",
		"varchar", "timestamptz", "numeric", "bytea",
		"integer[]", "text[]", "jsonb[]",
		"varchar(255)", "numeric(10,2)", "char(1)",
	}
	for _, typ := range known {
		assert.True(t, pgd.IsKnownBuiltinType(typ), "should be known: %s", typ)
	}

	unknown := []string{"foobar", "my_custom_type", ""}
	for _, typ := range unknown {
		assert.False(t, pgd.IsKnownBuiltinType(typ), "should be unknown: %s", typ)
	}
}

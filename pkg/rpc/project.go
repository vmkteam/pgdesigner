package rpc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/designer/lint"
	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
	"github.com/vmkteam/zenrpc/v2"
)

// ProjectService provides access to the loaded .pgd project.
type ProjectService struct {
	zenrpc.Service
	project *pgd.Project
	store   *store.ProjectStore // nil for read-only mode
}

// NewProjectService creates a read-only ProjectService.
func NewProjectService(project *pgd.Project) *ProjectService {
	return &ProjectService{project: project}
}

// NewProjectServiceWithStore creates a ProjectService backed by a ProjectStore (read-write).
func NewProjectServiceWithStore(s *store.ProjectStore) *ProjectService {
	return &ProjectService{project: s.Project(), store: s}
}

// GetInfo returns project metadata.
//
//zenrpc:return ProjectInfo
func (s ProjectService) GetInfo() ProjectInfo {
	var tables, refs, indexes int
	for _, sc := range s.project.Schemas {
		tables += len(sc.Tables)
		indexes += len(sc.Indexes)
		for _, t := range sc.Tables {
			refs += len(t.FKs)
		}
	}
	var autoSave bool
	if s.store != nil {
		autoSave = s.store.AutoSave()
	}
	schemaNames := MapV(s.project.Schemas, func(sc pgd.Schema) string { return sc.Name })
	return ProjectInfo{
		Name:            s.project.ProjectMeta.Name,
		PgVersion:       s.project.PgVersion,
		Tables:          tables,
		References:      refs,
		Indexes:         indexes,
		AutoSave:        autoSave,
		Schemas:         schemaNames,
		DefaultNullable: s.project.ProjectMeta.Settings.Defaults.Nullable != "false",
	}
}

// GetSchema returns the ERD schema for rendering in the frontend.
//
//zenrpc:return ERDSchema
func (s ProjectService) GetSchema() ERDSchema {
	return newERDSchema(s.project.ToERDSchema())
}

// GetDDL returns the full DDL for the project.
//
//zenrpc:return string
func (s ProjectService) GetDDL() string {
	return pgd.GenerateDDL(s.project)
}

// Lint validates the project and returns lint issues.
//
//zenrpc:return []LintIssue
func (s ProjectService) Lint() []LintIssue {
	return NewLintIssues(lint.Validate(s.project))
}

// ListObjects returns a flat list of all database objects for Go-To search.
//
//zenrpc:return []ObjectItem
func (s ProjectService) ListObjects() []ObjectItem {
	return newObjectItems(s.project)
}

// GetTable returns full table data for the Table Editor.
//
//zenrpc:name table name
//zenrpc:return TableDetail
//zenrpc:404 Not Found
func (s ProjectService) GetTable(name string) (*TableDetail, error) {
	// Support qualified name "schema.table" or plain "table"
	defaultSchema := s.project.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	for i := range s.project.Schemas {
		schema := &s.project.Schemas[i]
		for j := range schema.Tables {
			t := &schema.Tables[j]
			qualName := t.Name
			if schema.Name != defaultSchema {
				qualName = schema.Name + "." + t.Name
			}
			if qualName == name || t.Name == name {
				return newTableDetail(s.project, t, schema), nil
			}
		}
	}
	return nil, fmt.Errorf("table %q not found", name)
}

// SaveProject writes the project to the .pgd file.
//
//zenrpc:return bool
func (s ProjectService) SaveProject() (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	return true, s.store.Save()
}

// SaveLayout updates table positions in the default layout.
//
//zenrpc:positions table positions
//zenrpc:return bool
func (s ProjectService) SaveLayout(positions []LayoutPosition) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	entities := MapV(positions, func(p LayoutPosition) pgd.LayoutEntity {
		return pgd.LayoutEntity{Table: p.Name, Schema: p.Schema, X: p.X, Y: p.Y}
	})
	return true, s.store.UpdateLayout(entities)
}

// IsDirty reports whether the project has unsaved changes.
//
//zenrpc:return bool
func (s ProjectService) IsDirty() bool {
	if s.store == nil {
		return false
	}
	return s.store.IsDirty()
}

// GetAutoSave reports whether auto-save is enabled.
//
//zenrpc:return bool
func (s ProjectService) GetAutoSave() bool {
	if s.store == nil {
		return false
	}
	return s.store.AutoSave()
}

// SetAutoSave enables or disables auto-save after each mutation.
//
//zenrpc:enabled auto-save flag
//zenrpc:return bool
func (s ProjectService) SetAutoSave(enabled bool) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	s.store.SetAutoSave(enabled)
	return true, nil
}

// ListTypes returns available column types for autocomplete.
//
//zenrpc:return []TypeInfo
func (s ProjectService) ListTypes() []TypeInfo {
	var types []TypeInfo

	// Built-in PG18 types
	builtins := []struct{ name, category string }{
		{"bigint", "numeric"}, {"bigserial", "numeric"}, {"boolean", "boolean"},
		{"bytea", "binary"}, {"char", "character"}, {"character", "character"},
		{"character varying", "character"}, {"cidr", "network"}, {"circle", "geometric"},
		{"date", "datetime"}, {"double precision", "numeric"}, {"inet", "network"},
		{"integer", "numeric"}, {"interval", "datetime"}, {"json", "json"},
		{"jsonb", "json"}, {"line", "geometric"}, {"lseg", "geometric"},
		{"macaddr", "network"}, {"macaddr8", "network"}, {"money", "numeric"},
		{"numeric", "numeric"}, {"oid", "system"}, {"path", "geometric"},
		{"point", "geometric"}, {"polygon", "geometric"}, {"real", "numeric"},
		{"serial", "numeric"}, {"smallint", "numeric"}, {"smallserial", "numeric"},
		{"text", "character"}, {"time", "datetime"}, {"time with time zone", "datetime"},
		{"timestamp", "datetime"}, {"timestamptz", "datetime"}, {"tsquery", "search"},
		{"tsvector", "search"}, {"uuid", "other"}, {"varchar", "character"},
		{"xml", "other"},
	}
	for _, b := range builtins {
		types = append(types, TypeInfo{Name: b.name, Category: b.category, Source: "builtin"})
	}

	// Array forms of common types
	for _, name := range []string{"integer", "text", "varchar", "bigint", "boolean", "jsonb", "uuid"} {
		types = append(types, TypeInfo{Name: name + "[]", Category: "array", Source: "builtin"})
	}

	// User-defined types from the project
	if s.project.Types != nil {
		for _, e := range s.project.Types.Enums {
			types = append(types, TypeInfo{Name: e.Name, Category: "enum", Source: "user"})
		}
		for _, c := range s.project.Types.Composites {
			types = append(types, TypeInfo{Name: c.Name, Category: "composite", Source: "user"})
		}
		for _, d := range s.project.Types.Domains {
			types = append(types, TypeInfo{Name: d.Name, Category: "domain", Source: "user"})
		}
	}

	return types
}

// UpdateTable applies changes to a table. Each section is optional (null = skip).
//
//zenrpc:name      qualified table name
//zenrpc:general   table properties
//zenrpc:columns   full column list replacement
//zenrpc:pk        PK constraint (null name = remove)
//zenrpc:fks       full FK list replacement
//zenrpc:uniques   full UNIQUE list replacement
//zenrpc:checks    full CHECK list replacement
//zenrpc:excludes  full EXCLUDE list replacement
//zenrpc:indexes   full index list replacement
//zenrpc:return    TableDetail
func (s ProjectService) UpdateTable(
	name string,
	general *GeneralInput,
	columns []ColumnInput,
	pk *PKInput,
	fks []FKInput,
	uniques []UniqueInput,
	checks []CheckInput,
	excludes []ExcludeInput,
	indexes []IndexInput,
	partitionBy *PartitionByRPC,
	partitions []PartitionRPC,
) (*TableDetail, error) {
	if s.store == nil {
		return nil, errors.New("read-only mode")
	}

	if general != nil {
		if err := s.store.UpdateTableGeneral(name, general.Name, general.Comment, general.Unlogged, general.Generate); err != nil {
			return nil, err
		}
		// If renamed, use new name for subsequent lookups.
		if general.Name != nil {
			name = *general.Name
		}
	}

	if columns != nil {
		if err := s.store.UpdateTableColumns(name, MapV(columns, ColumnInput.toPGD)); err != nil {
			return nil, err
		}
	}

	if pk != nil {
		if err := s.store.UpdateTablePK(name, pk.toPGD()); err != nil {
			return nil, err
		}
	}

	if fks != nil {
		if err := s.store.UpdateTableFKs(name, MapV(fks, FKInput.toPGD)); err != nil {
			return nil, err
		}
	}

	if uniques != nil {
		if err := s.store.UpdateTableUniques(name, MapV(uniques, UniqueInput.toPGD)); err != nil {
			return nil, err
		}
	}

	if checks != nil {
		if err := s.store.UpdateTableChecks(name, MapV(checks, CheckInput.toPGD)); err != nil {
			return nil, err
		}
	}

	if excludes != nil {
		if err := s.store.UpdateTableExcludes(name, MapV(excludes, ExcludeInput.toPGD)); err != nil {
			return nil, err
		}
	}

	if indexes != nil {
		idxTable := name
		if i := strings.LastIndex(idxTable, "."); i >= 0 {
			idxTable = idxTable[i+1:]
		}
		pgdI := MapV(indexes, func(idx IndexInput) pgd.Index {
			idx.Table = idxTable
			return idx.toPGD()
		})
		if err := s.store.UpdateTableIndexes(name, pgdI); err != nil {
			return nil, err
		}
	}

	if err := applyPartitions(s.store, name, partitionBy, partitions); err != nil {
		return nil, err
	}

	// Validate the resulting table (server-side, Phase 2).
	issues := lint.ValidateTable(s.project, name, true)
	if len(issues) > 0 {
		return nil, &zenrpc.Error{
			Code:    422,
			Message: fmt.Sprintf("validation failed: %d error(s)", len(issues)),
			Data:    ValidationErrorData{Issues: NewLintIssues(issues)},
		}
	}

	// Re-read and return updated table.
	return s.GetTable(name)
}

// PreviewDiff returns ALTER SQL that would result from applying the given changes.
// It does NOT modify the project — only computes the diff.
//
//zenrpc:name      qualified table name
//zenrpc:general   table properties
//zenrpc:columns   full column list
//zenrpc:pk        PK constraint
//zenrpc:fks       full FK list
//zenrpc:uniques   full UNIQUE list
//zenrpc:checks    full CHECK list
//zenrpc:excludes  full EXCLUDE list
//zenrpc:indexes   full index list
//zenrpc:return    []DiffChange
func (s ProjectService) PreviewDiff(
	name string,
	general *GeneralInput,
	columns []ColumnInput,
	pk *PKInput,
	fks []FKInput,
	uniques []UniqueInput,
	checks []CheckInput,
	excludes []ExcludeInput,
	indexes []IndexInput,
) ([]DiffChange, error) {
	schema, table := s.store.FindTable(name)
	if table == nil {
		return nil, fmt.Errorf("table %q not found", name)
	}

	// Build "old" schema fragment from current state.
	oldSchema := newSchemaFragment(schema, table)

	// Build "new" table by applying changes.
	newTable := newTableCopy(table)
	applyGeneralToTable(&newTable, general)
	if columns != nil {
		newTable.Columns = MapV(columns, ColumnInput.toPGD)
	}
	if pk != nil {
		newTable.PK = pk.toPGD()
	}
	if fks != nil {
		newTable.FKs = MapV(fks, FKInput.toPGD)
	}
	if uniques != nil {
		newTable.Uniques = MapV(uniques, UniqueInput.toPGD)
	}
	if checks != nil {
		newTable.Checks = MapV(checks, CheckInput.toPGD)
	}
	if excludes != nil {
		newTable.Excludes = MapV(excludes, ExcludeInput.toPGD)
	}

	// Build "new" schema with indexes.
	newSchema := pgd.Schema{
		Name:   schema.Name,
		Tables: []pgd.Table{newTable},
	}
	if indexes != nil {
		tblName := table.Name
		newSchema.Indexes = MapV(indexes, func(idx IndexInput) pgd.Index {
			idx.Table = tblName
			return idx.toPGD()
		})
	} else {
		// Keep existing indexes for this table.
		for _, idx := range schema.Indexes {
			if idx.Table == table.Name {
				newSchema.Indexes = append(newSchema.Indexes, idx)
			}
		}
	}

	oldProject := &pgd.Project{Schemas: []pgd.Schema{oldSchema}}
	newProject := &pgd.Project{Schemas: []pgd.Schema{newSchema}}
	result := diff.Diff(oldProject, newProject)

	return NewDiffChanges(result.Changes), nil
}

// DiffUnsaved returns ALTER SQL for all unsaved changes (saved snapshot vs current state).
//
//zenrpc:return DiffUnsavedResult
func (s ProjectService) DiffUnsaved() (*DiffUnsavedResult, error) {
	if s.store == nil {
		return &DiffUnsavedResult{}, nil
	}
	saved := s.store.SavedProject()
	if saved == nil {
		return &DiffUnsavedResult{}, nil
	}
	result := diff.Diff(saved, s.project)
	return &DiffUnsavedResult{
		SQL:     result.SQL(),
		Changes: NewDiffChanges(result.Changes),
	}, nil
}

// FixLintIssues applies auto-fixes for selected lint issues.
//
//zenrpc:issues  selected issues (code + path pairs)
//zenrpc:return  FixLintResult
func (s ProjectService) FixLintIssues(issues []LintFixRequest) (*FixLintResult, error) {
	if s.store == nil {
		return nil, errors.New("read-only mode")
	}
	if len(issues) == 0 {
		return &FixLintResult{Issues: s.Lint()}, nil
	}

	// Validate once, index by code+path for O(1) lookup.
	current := lint.Validate(s.project)
	type key struct{ code, path string }
	idx := make(map[key]lint.Issue, len(current))
	for _, cur := range current {
		idx[key{cur.Code, cur.Path}] = cur
	}

	// Convert to lint.Issue, restoring full message (needed for W002/W015 FK name extraction).
	lintIssues := make([]lint.Issue, len(issues))
	for i, req := range issues {
		if cur, ok := idx[key{req.Code, req.Path}]; ok {
			lintIssues[i] = cur
		} else {
			lintIssues[i] = lint.Issue{Code: req.Code, Path: req.Path}
		}
	}

	var results []lint.FixResult
	if err := s.store.ApplyLintFixes(func(p *pgd.Project) {
		results = lint.Fix(p, lintIssues)
	}); err != nil {
		return nil, err
	}

	return &FixLintResult{
		Fixed:  len(results),
		Issues: s.Lint(), // re-validate
	}, nil
}

// IgnoreLintRules adds rules to project or table ignore list.
//
//zenrpc:rules  rule codes to ignore
//zenrpc:table  optional qualified table name (null = project level)
//zenrpc:return []LintIssue
func (s ProjectService) IgnoreLintRules(rules []string, table *string) ([]LintIssue, error) {
	if s.store == nil {
		return nil, errors.New("read-only mode")
	}
	if err := s.store.AddIgnoreRules(rules, table); err != nil {
		return nil, err
	}
	return s.Lint(), nil // re-validate with updated ignores
}

// GetIgnoredRules returns all ignored lint rules from project and table settings.
//
//zenrpc:return []IgnoredRule
func (s ProjectService) GetIgnoredRules() []IgnoredRule {
	var result []IgnoredRule
	// Project-level
	if s.project.ProjectMeta.Settings.Lint != nil {
		for _, code := range splitCSV(s.project.ProjectMeta.Settings.Lint.IgnoreRules) {
			result = append(result, IgnoredRule{Code: code, Title: ruleTitle(code), Scope: "project"})
		}
	}
	// Table-level
	for _, schema := range s.project.Schemas {
		for _, t := range schema.Tables {
			for _, code := range splitCSV(t.LintIgnore) {
				result = append(result, IgnoredRule{Code: code, Title: ruleTitle(code), Scope: schema.Name + "." + t.Name})
			}
		}
	}
	return result
}

// UnignoreLintRules removes rules from project or table ignore list.
//
//zenrpc:rules  codes to unignore
//zenrpc:table  optional table name (null = project level)
//zenrpc:return bool
func (s ProjectService) UnignoreLintRules(rules []string, table *string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.RemoveIgnoreRules(rules, table); err != nil {
		return false, err
	}
	return true, nil
}

// CreateTable creates a new empty table in the specified schema.
//
//zenrpc:schemaName schema name
//zenrpc:tableName  table name
//zenrpc:return     bool
func (s ProjectService) CreateTable(schemaName, tableName string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.CreateTable(schemaName, tableName); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteTable removes a table and its indexes from the project.
//
//zenrpc:name qualified table name
//zenrpc:return bool
func (s ProjectService) DeleteTable(name string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.DeleteTable(name); err != nil {
		return false, err
	}
	return true, nil
}

// CreateSchema adds a new empty schema to the project.
//
//zenrpc:name schema name
//zenrpc:return bool
func (s ProjectService) CreateSchema(name string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.CreateSchema(name); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteSchema removes an empty schema from the project.
//
//zenrpc:name schema name
//zenrpc:return bool
func (s ProjectService) DeleteSchema(name string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.DeleteSchema(name); err != nil {
		return false, err
	}
	return true, nil
}

// MoveTable transfers a table from its current schema to another.
//
//zenrpc:name     qualified table name
//zenrpc:toSchema destination schema name
//zenrpc:return   bool
func (s ProjectService) MoveTable(name, toSchema string) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.MoveTable(name, toSchema); err != nil {
		return false, err
	}
	return true, nil
}

// GetProjectSettings returns editable project settings.
//
//zenrpc:return ProjectSettings
func (s ProjectService) GetProjectSettings() ProjectSettings {
	p := s.project
	var lintIgnore string
	if p.ProjectMeta.Settings.Lint != nil {
		lintIgnore = p.ProjectMeta.Settings.Lint.IgnoreRules
	}
	return ProjectSettings{
		Name:             p.ProjectMeta.Name,
		Description:      p.ProjectMeta.Description,
		PgVersion:        p.PgVersion,
		DefaultSchema:    p.DefaultSchema,
		NamingConvention: p.ProjectMeta.Settings.Naming.Convention,
		NamingTables:     p.ProjectMeta.Settings.Naming.Tables,
		DefaultNullable:  p.ProjectMeta.Settings.Defaults.Nullable,
		DefaultOnDelete:  p.ProjectMeta.Settings.Defaults.OnDelete,
		DefaultOnUpdate:  p.ProjectMeta.Settings.Defaults.OnUpdate,
		LintIgnoreRules:  lintIgnore,
	}
}

// UpdateProjectSettings saves project-level settings.
//
//zenrpc:settings project settings
//zenrpc:return bool
func (s ProjectService) UpdateProjectSettings(settings ProjectSettings) (bool, error) {
	if s.store == nil {
		return false, errors.New("read-only mode")
	}
	if err := s.store.UpdateProjectSettings(store.ProjectSettingsInput{
		Name:             settings.Name,
		Description:      settings.Description,
		PgVersion:        settings.PgVersion,
		DefaultSchema:    settings.DefaultSchema,
		NamingConvention: settings.NamingConvention,
		NamingTables:     settings.NamingTables,
		DefaultNullable:  settings.DefaultNullable,
		DefaultOnDelete:  settings.DefaultOnDelete,
		DefaultOnUpdate:  settings.DefaultOnUpdate,
		LintIgnoreRules:  settings.LintIgnoreRules,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// LintTable validates a single table and returns all lint issues.
//
//zenrpc:name table name
//zenrpc:return []LintIssue
//zenrpc:404 Not Found
func (s ProjectService) LintTable(name string) ([]LintIssue, error) {
	issues := lint.ValidateTable(s.project, name, false)
	if issues == nil {
		return nil, fmt.Errorf("table %q not found", name)
	}
	return NewLintIssues(issues), nil
}

package store

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// ProjectStore manages the in-memory project and its persistence to disk.
// It keeps a saved snapshot for diffing unsaved changes.
type ProjectStore struct {
	mu           sync.RWMutex
	saved        *pgd.Project // snapshot: state at last load/save
	project      *pgd.Project // working copy: all mutations go here
	filePath     string
	dirty        bool
	autoSave     bool
	backupTicker *time.Ticker
	stopBackup   chan struct{}
}

// NewProjectStore creates a store for the given project and file path.
func NewProjectStore(project *pgd.Project, filePath string) *ProjectStore {
	return &ProjectStore{
		saved:    deepCopyProject(project),
		project:  project,
		filePath: filePath,
	}
}

// deepCopyProject creates an independent deep copy via XML round-trip.
func deepCopyProject(p *pgd.Project) *pgd.Project {
	data, err := xml.Marshal(p)
	if err != nil {
		return nil
	}
	var cp pgd.Project
	if err := xml.Unmarshal(data, &cp); err != nil {
		return nil
	}
	return &cp
}

// Project returns the in-memory project (read-only). Caller must not mutate.
func (s *ProjectStore) Project() *pgd.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.project
}

// SavedProject returns the last-saved project snapshot (read-only).
func (s *ProjectStore) SavedProject() *pgd.Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saved
}

// IsDirty reports whether there are unsaved changes.
func (s *ProjectStore) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// AutoSave reports whether auto-save is enabled.
func (s *ProjectStore) AutoSave() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoSave
}

// SetAutoSave enables or disables auto-save after each mutation.
func (s *ProjectStore) SetAutoSave(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoSave = enabled
}

// FilePath returns the current .pgd file path.
func (s *ProjectStore) FilePath() string {
	return s.filePath
}

// Save writes the project to the .pgd file and removes the .bak file.
func (s *ProjectStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *ProjectStore) saveLocked() error {
	if s.filePath == "" {
		return errors.New("no file path set")
	}
	if err := writeProjectFile(s.project, s.filePath); err != nil {
		return err
	}
	s.dirty = false
	s.saved = deepCopyProject(s.project)
	// Remove backup — the .pgd is now up to date.
	os.Remove(s.filePath + ".bak") //nolint:errcheck
	return nil
}

// SaveAs writes the project to a new file path and updates the store path.
func (s *ProjectStore) SaveAs(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := writeProjectFile(s.project, path); err != nil {
		return err
	}
	s.filePath = path
	s.dirty = false
	s.saved = deepCopyProject(s.project)
	return nil
}

// SaveBackup writes the current dirty state to .pgd.bak.
func (s *ProjectStore) SaveBackup() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.dirty || s.filePath == "" {
		return nil
	}
	return writeProjectFile(s.project, s.filePath+".bak")
}

// StartAutoBackup starts a goroutine that periodically writes .pgd.bak if dirty.
func (s *ProjectStore) StartAutoBackup(interval time.Duration) {
	s.stopBackup = make(chan struct{})
	s.backupTicker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.backupTicker.C:
				s.SaveBackup() //nolint:errcheck
			case <-s.stopBackup:
				return
			}
		}
	}()
}

// StopAutoBackup stops the periodic backup goroutine.
func (s *ProjectStore) StopAutoBackup() {
	if s.backupTicker != nil {
		s.backupTicker.Stop()
	}
	if s.stopBackup != nil {
		close(s.stopBackup)
	}
}

// markDirtyLocked marks the project as dirty and auto-saves if enabled. Must be called with mu held.
func (s *ProjectStore) markDirtyLocked() error {
	s.dirty = true
	if s.autoSave {
		return s.saveLocked()
	}
	return nil
}

// --- Mutations ---

// FindTable returns the schema and table by qualified name ("schema.table" or "table").
func (s *ProjectStore) FindTable(name string) (*pgd.Schema, *pgd.Table) {
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
				return schema, t
			}
		}
	}
	return nil, nil
}

// UpdateTableColumns replaces all columns in a table.
func (s *ProjectStore) UpdateTableColumns(name string, columns []pgd.Column) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.Columns = columns
	return s.markDirtyLocked()
}

// UpdateTablePK sets or removes the PK on a table.
func (s *ProjectStore) UpdateTablePK(name string, pk *pgd.PrimaryKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.PK = pk
	return s.markDirtyLocked()
}

// UpdateTableFKs replaces all FKs on a table.
func (s *ProjectStore) UpdateTableFKs(name string, fks []pgd.ForeignKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.FKs = fks
	return s.markDirtyLocked()
}

// UpdateTableUniques replaces all UNIQUE constraints on a table.
func (s *ProjectStore) UpdateTableUniques(name string, uniques []pgd.Unique) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.Uniques = uniques
	return s.markDirtyLocked()
}

// UpdateTableChecks replaces all CHECK constraints on a table.
func (s *ProjectStore) UpdateTableChecks(name string, checks []pgd.Check) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.Checks = checks
	return s.markDirtyLocked()
}

// UpdateTableExcludes replaces all exclude constraints for a table.
func (s *ProjectStore) UpdateTableExcludes(name string, excludes []pgd.Exclude) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.Excludes = excludes
	return s.markDirtyLocked()
}

// UpdateTableIndexes replaces all indexes for a table in its schema.
func (s *ProjectStore) UpdateTableIndexes(name string, indexes []pgd.Index) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	// Remove old indexes for this table, add new ones.
	var kept []pgd.Index
	for _, idx := range schema.Indexes {
		if idx.Table != t.Name {
			kept = append(kept, idx)
		}
	}
	kept = append(kept, indexes...)
	schema.Indexes = kept
	return s.markDirtyLocked()
}

// UpdateTableGeneral updates table-level properties (name, comment, unlogged, generate).
func (s *ProjectStore) UpdateTableGeneral(name string, newName, comment *string, unlogged, generate *bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	schema, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	if newName != nil && *newName != t.Name {
		oldName := t.Name
		t.Name = *newName
		// Update index references to the renamed table.
		for i := range schema.Indexes {
			if schema.Indexes[i].Table == oldName {
				schema.Indexes[i].Table = *newName
			}
		}
		// Update FK references from other tables pointing to the renamed table.
		for si := range s.project.Schemas {
			for ti := range s.project.Schemas[si].Tables {
				for fi := range s.project.Schemas[si].Tables[ti].FKs {
					if s.project.Schemas[si].Tables[ti].FKs[fi].ToTable == oldName {
						s.project.Schemas[si].Tables[ti].FKs[fi].ToTable = *newName
					}
				}
			}
		}
		// Update layout entity references.
		for li := range s.project.Layouts.Layouts {
			for ei := range s.project.Layouts.Layouts[li].Entities {
				if s.project.Layouts.Layouts[li].Entities[ei].Table == oldName {
					s.project.Layouts.Layouts[li].Entities[ei].Table = *newName
				}
			}
		}
	}
	if comment != nil {
		t.Comment = *comment
	}
	if unlogged != nil {
		if *unlogged {
			t.Unlogged = "true"
		} else {
			t.Unlogged = ""
		}
	}
	if generate != nil {
		if *generate {
			t.Generate = ""
		} else {
			t.Generate = "false"
		}
	}
	return s.markDirtyLocked()
}

// CreateTable adds a new empty table with an id column and PK.
func (s *ProjectStore) CreateTable(schemaName, tableName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.project.Schemas {
		if s.project.Schemas[i].Name == schemaName {
			s.project.Schemas[i].Tables = append(s.project.Schemas[i].Tables, pgd.Table{
				Name: tableName,
				Columns: []pgd.Column{
					{Name: "id", Type: "integer", Nullable: "false"},
				},
				PK: &pgd.PrimaryKey{
					Name:    "pk_" + tableName,
					Columns: []pgd.ColRef{{Name: "id"}},
				},
			})
			return s.markDirtyLocked()
		}
	}
	return fmt.Errorf("schema %q not found", schemaName)
}

// DeleteTable removes a table and its indexes from the project.
func (s *ProjectStore) DeleteTable(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.project.Schemas {
		schema := &s.project.Schemas[i]
		for j := range schema.Tables {
			t := &schema.Tables[j]
			qualName := t.Name
			if schema.Name != s.defaultSchemaLocked() {
				qualName = schema.Name + "." + t.Name
			}
			if qualName == name || t.Name == name {
				tblName := t.Name
				// Remove table
				schema.Tables = append(schema.Tables[:j], schema.Tables[j+1:]...)
				// Remove its indexes
				var kept []pgd.Index
				for _, idx := range schema.Indexes {
					if idx.Table != tblName {
						kept = append(kept, idx)
					}
				}
				schema.Indexes = kept
				return s.markDirtyLocked()
			}
		}
	}
	return fmt.Errorf("table %q not found", name)
}

// CreateSchema adds a new empty schema to the project.
func (s *ProjectStore) CreateSchema(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, schema := range s.project.Schemas {
		if schema.Name == name {
			return fmt.Errorf("schema %q already exists", name)
		}
	}
	s.project.Schemas = append(s.project.Schemas, pgd.Schema{Name: name})
	return s.markDirtyLocked()
}

// DeleteSchema removes an empty schema from the project.
func (s *ProjectStore) DeleteSchema(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, schema := range s.project.Schemas {
		if schema.Name == name {
			if len(schema.Tables) > 0 {
				return fmt.Errorf("schema %q is not empty (%d tables)", name, len(schema.Tables))
			}
			s.project.Schemas = append(s.project.Schemas[:i], s.project.Schemas[i+1:]...)
			return s.markDirtyLocked()
		}
	}
	return fmt.Errorf("schema %q not found", name)
}

// MoveTable transfers a table (with its indexes) from one schema to another.
func (s *ProjectStore) MoveTable(name, toSchema string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find destination schema.
	var dst *pgd.Schema
	for i := range s.project.Schemas {
		if s.project.Schemas[i].Name == toSchema {
			dst = &s.project.Schemas[i]
			break
		}
	}
	if dst == nil {
		return fmt.Errorf("schema %q not found", toSchema)
	}

	// Find source table.
	var src *pgd.Schema
	var table pgd.Table
	var tableIndexes []pgd.Index
	found := false
	for i := range s.project.Schemas {
		schema := &s.project.Schemas[i]
		for j := range schema.Tables {
			t := &schema.Tables[j]
			qualName := t.Name
			if schema.Name != s.defaultSchemaLocked() {
				qualName = schema.Name + "." + t.Name
			}
			if qualName == name || t.Name == name {
				if schema.Name == toSchema {
					return fmt.Errorf("table %q is already in schema %q", name, toSchema)
				}
				src = schema
				table = *t
				// Collect indexes for this table.
				var kept []pgd.Index
				for _, idx := range schema.Indexes {
					if idx.Table == t.Name {
						tableIndexes = append(tableIndexes, idx)
					} else {
						kept = append(kept, idx)
					}
				}
				schema.Indexes = kept
				schema.Tables = append(schema.Tables[:j], schema.Tables[j+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return fmt.Errorf("table %q not found", name)
	}

	// Add to destination.
	dst.Tables = append(dst.Tables, table)
	dst.Indexes = append(dst.Indexes, tableIndexes...)

	// Update layout entity schema references.
	for li := range s.project.Layouts.Layouts {
		for ei := range s.project.Layouts.Layouts[li].Entities {
			e := &s.project.Layouts.Layouts[li].Entities[ei]
			if e.Table == table.Name && e.Schema == src.Name {
				e.Schema = toSchema
			}
		}
	}

	return s.markDirtyLocked()
}

func (s *ProjectStore) defaultSchemaLocked() string {
	if s.project.DefaultSchema != "" {
		return s.project.DefaultSchema
	}
	return "public"
}

// ApplyLintFixes runs a fix function on the project under write lock.
func (s *ProjectStore) ApplyLintFixes(fn func(p *pgd.Project)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.project)
	return s.markDirtyLocked()
}

// AddIgnoreRules adds lint ignore rules at project or table level.
// If tableName is provided but table is not found, falls through to project-level.
func (s *ProjectStore) AddIgnoreRules(rules []string, tableName *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tableName != nil {
		if _, t := s.FindTable(*tableName); t != nil {
			t.LintIgnore = appendCSV(t.LintIgnore, rules)
			return s.markDirtyLocked()
		}
	}
	// Project-level ignore
	if s.project.ProjectMeta.Settings.Lint == nil {
		s.project.ProjectMeta.Settings.Lint = &pgd.Lint{}
	}
	s.project.ProjectMeta.Settings.Lint.IgnoreRules = appendCSV(s.project.ProjectMeta.Settings.Lint.IgnoreRules, rules)
	return s.markDirtyLocked()
}

func appendCSV(existing string, rules []string) string {
	for _, r := range rules {
		if !containsRule(existing, r) {
			if existing != "" {
				existing += ","
			}
			existing += r
		}
	}
	return existing
}

// RemoveIgnoreRules removes lint ignore rules from project or table level.
func (s *ProjectStore) RemoveIgnoreRules(rules []string, tableName *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	remove := make(map[string]bool, len(rules))
	for _, r := range rules {
		remove[r] = true
	}

	if tableName != nil {
		_, t := s.FindTable(*tableName)
		if t == nil {
			return fmt.Errorf("table %q not found", *tableName)
		}
		t.LintIgnore = filterCSV(t.LintIgnore, remove)
	} else if s.project.ProjectMeta.Settings.Lint != nil {
		s.project.ProjectMeta.Settings.Lint.IgnoreRules = filterCSV(s.project.ProjectMeta.Settings.Lint.IgnoreRules, remove)
	}
	return s.markDirtyLocked()
}

func filterCSV(csv string, remove map[string]bool) string {
	var kept []string
	for _, r := range strings.Split(csv, ",") {
		r = strings.TrimSpace(r)
		if r != "" && !remove[r] {
			kept = append(kept, r)
		}
	}
	return strings.Join(kept, ",")
}

func containsRule(csv, rule string) bool {
	for _, r := range strings.Split(csv, ",") {
		if strings.TrimSpace(r) == rule {
			return true
		}
	}
	return false
}

// UpdateLayout replaces all entity positions in the default layout.
func (s *ProjectStore) UpdateLayout(entities []pgd.LayoutEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.project.Layouts.Layouts) == 0 {
		s.project.Layouts.Layouts = []pgd.Layout{{Name: "Default", Default: "true"}}
	}
	s.project.Layouts.Layouts[0].Entities = entities
	return s.markDirtyLocked()
}

// UpdateTablePartitions replaces partition-by and partitions for a table.
func (s *ProjectStore) UpdateTablePartitions(name string, partitionBy *pgd.PartitionBy, partitions []pgd.Partition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.FindTable(name)
	if t == nil {
		return fmt.Errorf("table %q not found", name)
	}
	t.PartitionBy = partitionBy
	t.Partitions = partitions
	return s.markDirtyLocked()
}

// ProjectSettingsInput holds project-level metadata and settings for UpdateProjectSettings.
type ProjectSettingsInput struct {
	Name             string
	Description      string
	PgVersion        string
	DefaultSchema    string
	NamingConvention string
	NamingTables     string
	DefaultNullable  string
	DefaultOnDelete  string
	DefaultOnUpdate  string
	LintIgnoreRules  string
}

// UpdateProjectSettings updates project-level metadata and settings.
func (s *ProjectStore) UpdateProjectSettings(in ProjectSettingsInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.project.ProjectMeta.Name = in.Name
	s.project.ProjectMeta.Description = in.Description
	s.project.PgVersion = in.PgVersion
	s.project.DefaultSchema = in.DefaultSchema
	s.project.ProjectMeta.Settings.Naming.Convention = in.NamingConvention
	s.project.ProjectMeta.Settings.Naming.Tables = in.NamingTables
	s.project.ProjectMeta.Settings.Defaults.Nullable = in.DefaultNullable
	s.project.ProjectMeta.Settings.Defaults.OnDelete = in.DefaultOnDelete
	s.project.ProjectMeta.Settings.Defaults.OnUpdate = in.DefaultOnUpdate
	if in.LintIgnoreRules != "" {
		if s.project.ProjectMeta.Settings.Lint == nil {
			s.project.ProjectMeta.Settings.Lint = &pgd.Lint{}
		}
		s.project.ProjectMeta.Settings.Lint.IgnoreRules = in.LintIgnoreRules
	} else if s.project.ProjectMeta.Settings.Lint != nil {
		s.project.ProjectMeta.Settings.Lint.IgnoreRules = ""
	}
	return s.markDirtyLocked()
}

// --- File I/O ---

func writeProjectFile(p *pgd.Project, path string) error {
	out, err := xml.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal XML: %w", err)
	}
	return os.WriteFile(path, []byte(xml.Header+string(out)+"\n"), 0644)
}

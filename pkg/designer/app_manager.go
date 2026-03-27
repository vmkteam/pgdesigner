package designer

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	diffdemo "github.com/vmkteam/pgdesigner/demo/diff"
	pgddemo "github.com/vmkteam/pgdesigner/demo/schemas/pgd"
	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/format"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// blockedDirs contains directories that should not be listed for security.
var blockedDirs = map[string]bool{
	"/proc": true,
	"/sys":  true,
	"/dev":  true,
}

// DemoSchemaInfo describes an available embedded demo schema.
type DemoSchemaInfo struct {
	Name   string
	Title  string
	Tables int
	FKs    int
}

// demoSchemas is the hardcoded catalog of embedded demo schemas.
var demoSchemas = []DemoSchemaInfo{
	{Name: "chinook", Title: "Chinook", Tables: 11, FKs: 11},
	{Name: "northwind", Title: "Northwind", Tables: 14, FKs: 13},
	{Name: "pagila", Title: "Pagila", Tables: 15, FKs: 18},
	{Name: "airlines", Title: "Airlines", Tables: 8, FKs: 8},
	{Name: "adventureworks", Title: "AdventureWorks", Tables: 68, FKs: 89},
}

// DiffExampleInfo describes an available pre-built diff example.
type DiffExampleInfo struct {
	Name        string
	Title       string
	Description string
}

// diffExamples is the hardcoded catalog of embedded diff examples.
var diffExamples = []DiffExampleInfo{
	{Name: "add-column", Title: "Add Column", Description: "Add varchar NOT NULL column with default"},
	{Name: "add-table", Title: "Add Table", Description: "Create table + index + 2 FK"},
	{Name: "move-column", Title: "Move Column", Description: "Drop column from one table, add to another (DELETES_DATA)"},
	{Name: "modify-index", Title: "Modify Index", Description: "Change index columns + add WHERE predicate"},
}

// DirListing holds the result of listing a directory.
type DirListing struct {
	Path    string
	Entries []DirEntryInfo
}

// DirEntryInfo represents a file or directory in a directory listing.
type DirEntryInfo struct {
	Name      string
	IsDir     bool
	Size      int64
	ModTime   time.Time
	Supported bool
}

// RecentFileInfo holds metadata about a recently opened file.
type RecentFileInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
	Exists  bool
}

// DiffExampleResult holds the result of running a diff example.
type DiffExampleResult struct {
	Changes []diff.Change
	SQL     string
}

// AppManager handles project-level business operations:
// file open/close, demos, diff examples, directory browsing.
type AppManager struct{}

// NewAppManager creates a new AppManager.
func NewAppManager() *AppManager {
	return &AppManager{}
}

// OpenFile loads a project from a file path or DSN.
// Returns the project and the resolved .pgd file path.
func (m *AppManager) OpenFile(path string) (*pgd.Project, string, error) {
	project, err := format.LoadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("loading %s: %w", path, err)
	}
	return project, pgdFilePath(path), nil
}

// IntrospectDSN connects to a PostgreSQL database and returns a lightweight preview.
func (m *AppManager) IntrospectDSN(dsn string) (*pgre.PreviewResult, error) {
	return pgre.Preview(dsn)
}

// ImportDSNOptions controls what to import from a PostgreSQL database.
type ImportDSNOptions struct {
	Schemas    []string // schemas to import (empty = all)
	Tables     []string // "schema.table" to import (empty = all in selected schemas)
	Categories []string // object categories: views, matviews, functions, triggers, enums, domains, sequences, extensions
}

// ImportDSN imports a schema from PostgreSQL with filtering.
func (m *AppManager) ImportDSN(dsn string, opts ImportDSNOptions) (*pgd.Project, error) {
	cats := toSet(opts.Categories)
	full := cats["views"] || cats["matviews"] || cats["functions"] || cats["triggers"] || cats["enums"] || cats["domains"]

	project, err := format.LoadFile(dsn,
		format.WithSchemas(opts.Schemas...),
		format.WithFull(full),
	)
	if err != nil {
		return nil, fmt.Errorf("importing from DSN: %w", err)
	}

	filterProject(project, opts, cats)
	return project, nil
}

// filterProject removes unselected objects from the project.
func filterProject(p *pgd.Project, opts ImportDSNOptions, cats map[string]bool) {
	// Filter tables if specific list given
	if len(opts.Tables) > 0 {
		selected := toSet(opts.Tables)
		for i := range p.Schemas {
			tables := p.Schemas[i].Tables[:0]
			for _, t := range p.Schemas[i].Tables {
				if selected[p.Schemas[i].Name+"."+t.Name] {
					tables = append(tables, t)
				}
			}
			p.Schemas[i].Tables = tables
		}
	}

	// Remove empty schemas (no tables left)
	schemas := p.Schemas[:0]
	for _, s := range p.Schemas {
		if len(s.Tables) > 0 {
			schemas = append(schemas, s)
		}
	}
	p.Schemas = schemas

	// Filter categories
	if !cats["views"] && !cats["matviews"] {
		p.Views = nil
	} else if p.Views != nil {
		if !cats["views"] {
			p.Views.Views = nil
		}
		if !cats["matviews"] {
			p.Views.MatViews = nil
		}
	}
	if !cats["functions"] {
		p.Functions = nil
	}
	if !cats["triggers"] {
		p.Triggers = nil
	}
	if !cats["sequences"] {
		p.Sequences = nil
	}
	if !cats["extensions"] {
		p.Extensions = nil
	}
	if p.Types != nil {
		if !cats["enums"] {
			p.Types.Enums = nil
		}
		if !cats["domains"] {
			p.Types.Domains = nil
		}
	}
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

// OpenDemo loads an embedded demo schema by name.
func (m *AppManager) OpenDemo(name string) (*pgd.Project, error) {
	project, err := loadDemoSchema(name)
	if err != nil {
		return nil, fmt.Errorf("loading demo %s: %w", name, err)
	}
	return project, nil
}

// NewProject creates a new empty project.
func (m *AppManager) NewProject() *pgd.Project {
	return pgd.NewEmptyProject()
}

// ListDemoSchemas returns available embedded demo schemas.
func (m *AppManager) ListDemoSchemas() []DemoSchemaInfo {
	return demoSchemas
}

// ListDiffExamples returns available pre-built diff examples.
func (m *AppManager) ListDiffExamples() []DiffExampleInfo {
	return diffExamples
}

// RunDiffExample loads a diff pair and returns the diff result.
func (m *AppManager) RunDiffExample(name string) (*DiffExampleResult, error) {
	oldProject, err := loadDiffExample(name, "old.pgd")
	if err != nil {
		return nil, fmt.Errorf("loading old: %w", err)
	}
	newProject, err := loadDiffExample(name, "new.pgd")
	if err != nil {
		return nil, fmt.Errorf("loading new: %w", err)
	}
	result := diff.Diff(oldProject, newProject)
	return &DiffExampleResult{
		SQL:     result.SQL(),
		Changes: result.Changes,
	}, nil
}

// GetHomePath returns the user's home directory path.
func (m *AppManager) GetHomePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/"
	}
	return home
}

// ListDirectory lists files and subdirectories at the given path.
func (m *AppManager) ListDirectory(path string, showAll bool) (*DirListing, error) {
	path = expandHome(path)

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	for blocked := range blockedDirs {
		if path == blocked || strings.HasPrefix(path, blocked+"/") {
			return &DirListing{Path: path}, nil
		}
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var dirs, files []DirEntryInfo
	for _, e := range dirEntries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if e.IsDir() {
			dirs = append(dirs, DirEntryInfo{
				Name:    name,
				IsDir:   true,
				ModTime: info.ModTime(),
			})
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		supported := format.SupportedExtensions[ext]
		if !showAll && !supported {
			continue
		}

		files = append(files, DirEntryInfo{
			Name:      name,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Supported: supported,
		})
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })

	entries := make([]DirEntryInfo, 0, len(dirs)+len(files))
	entries = append(entries, dirs...)
	entries = append(entries, files...)

	return &DirListing{Path: path, Entries: entries}, nil
}

// GetRecentFilesInfo returns metadata for the given file paths.
func (m *AppManager) GetRecentFilesInfo(paths []string) []RecentFileInfo {
	result := make([]RecentFileInfo, 0, len(paths))
	for _, p := range paths {
		rf := RecentFileInfo{
			Path: p,
			Name: filepath.Base(p),
		}
		info, err := os.Stat(p)
		if err == nil {
			rf.Size = info.Size()
			rf.ModTime = info.ModTime()
			rf.Exists = true
		}
		result = append(result, rf)
	}
	return result
}

// loadDemoSchema loads an embedded demo schema by name.
func loadDemoSchema(name string) (*pgd.Project, error) {
	data, err := pgddemo.FS.ReadFile(name + ".pgd")
	if err != nil {
		return nil, err
	}
	var p pgd.Project
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// loadDiffExample loads a diff example file from embedded FS.
func loadDiffExample(name, file string) (*pgd.Project, error) {
	data, err := diffdemo.FS.ReadFile(name + "/" + file)
	if err != nil {
		return nil, err
	}
	var p pgd.Project
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// pgdFilePath returns the .pgd output path for a given input.
func pgdFilePath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".pgd":
		return path
	case ".pdd", ".dbs", ".dm2", ".sql":
		return strings.TrimSuffix(path, ext) + ".pgd"
	}
	return ""
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

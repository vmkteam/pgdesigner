package rpc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	diffdemo "github.com/vmkteam/pgdesigner/demo/diff"
	pgddemo "github.com/vmkteam/pgdesigner/demo/schemas/pgd"
	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/pgdesigner/pkg/format"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
	"github.com/vmkteam/zenrpc/v2"
)

const quitGracePeriod = 3 * time.Second

// demoSchemas is the hardcoded catalog of embedded demo schemas.
var demoSchemas = []DemoSchema{
	{Name: "chinook", Title: "Chinook", Tables: 11, FKs: 11},
	{Name: "northwind", Title: "Northwind", Tables: 14, FKs: 13},
	{Name: "pagila", Title: "Pagila", Tables: 15, FKs: 18},
	{Name: "airlines", Title: "Airlines", Tables: 8, FKs: 8},
	{Name: "adventureworks", Title: "AdventureWorks", Tables: 68, FKs: 89},
}

// ConfigCallbacks provides access to app config without circular imports.
type ConfigCallbacks struct {
	Register       func(email string) error
	IsRegistered   func() bool
	GetRecentFiles func() []string
	AddRecentFile  func(path string) error
}

// AppService provides application lifecycle methods.
type AppService struct {
	zenrpc.Service
	quitCh chan struct{}
	store  *store.ProjectStore
	config ConfigCallbacks
	mu     sync.Mutex
	timer  *time.Timer
}

// NewAppService creates an AppService that signals quit via the provided channel.
func NewAppService(quitCh chan struct{}, s *store.ProjectStore, cfg ConfigCallbacks) *AppService {
	return &AppService{quitCh: quitCh, store: s, config: cfg}
}

// Quit starts a delayed shutdown. If Ping is not called within the grace period, the server exits.
//
// zenrpc
func (s *AppService) Quit() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Already closed.
	select {
	case <-s.quitCh:
		return
	default:
	}

	if s.timer != nil {
		s.timer.Reset(quitGracePeriod)
		return
	}

	s.timer = time.AfterFunc(quitGracePeriod, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		select {
		case <-s.quitCh:
		default:
			close(s.quitCh)
		}
	})
}

// Ping cancels a pending shutdown (e.g. after page reload).
//
// zenrpc
func (s *AppService) Ping() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	return "pong"
}

func vcsVersion() string {
	result := "dev"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}
	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			result = v.Value
		}
	}
	if len(result) > 8 {
		result = result[:8]
	}
	return result
}

// About returns application metadata.
//
//zenrpc:return AboutInfo
func (s *AppService) About() AboutInfo {
	return AboutInfo{
		Name:        "PgDesigner",
		Description: "Visual PostgreSQL Schema Designer",
		Version:     vcsVersion(),
		GoVersion:   runtime.Version(),
		Target:      "PostgreSQL 18",
		Author:      "Sergey Bykov (sergeyfast)",
		License:     "PolyForm Noncommercial 1.0.0",
		Website:     "https://pgdesigner.io",
		GitHub:      "https://github.com/vmkteam/pgdesigner",
	}
}

// ListDemoSchemas returns available embedded demo schemas.
//
//zenrpc:return []DemoSchema
func (s *AppService) ListDemoSchemas() []DemoSchema {
	return demoSchemas
}

// OpenDemo loads an embedded demo schema by name.
//
//zenrpc:name demo schema name (chinook, northwind, pagila, airlines, adventureworks)
//zenrpc:return bool
func (s *AppService) OpenDemo(name string) (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	project, err := loadDemoSchema(name)
	if err != nil {
		return false, fmt.Errorf("loading demo %s: %w", name, err)
	}
	s.store.ReplaceProject(project, "")
	s.store.SetDemo(true)
	return true, nil
}

// OpenFile opens a file by path, auto-converting if necessary.
//
//zenrpc:path full path to .pgd, .pdd, .dbs, .dm2, .sql file or PostgreSQL DSN
//zenrpc:return bool
func (s *AppService) OpenFile(path string) (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	project, err := format.LoadFile(path)
	if err != nil {
		return false, fmt.Errorf("loading %s: %w", path, err)
	}
	fp := pgdFilePath(path)
	s.store.ReplaceProject(project, fp)
	if s.config.AddRecentFile != nil {
		_ = s.config.AddRecentFile(fp)
	}
	return true, nil
}

// NewProject creates a new empty project, replacing the current one.
//
//zenrpc:return bool
func (s *AppService) NewProject() (bool, error) { return s.resetProject() }

// CloseProject replaces current project with empty one (returns to welcome screen).
//
//zenrpc:return bool
func (s *AppService) CloseProject() (bool, error) { return s.resetProject() }

func (s *AppService) resetProject() (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	s.store.ReplaceProject(pgd.NewEmptyProject(), "")
	return true, nil
}

// Register sets the registered email (honor system, no validation).
//
//zenrpc:email registered email
//zenrpc:return bool
func (s *AppService) Register(email string) (bool, error) {
	if s.config.Register == nil {
		return false, errors.New("registration not available")
	}
	if err := s.config.Register(email); err != nil {
		return false, err
	}
	return true, nil
}

// GetRecentFiles returns the list of recently opened files.
//
//zenrpc:return []string
func (s *AppService) GetRecentFiles() []string {
	if s.config.GetRecentFiles == nil {
		return nil
	}
	return s.config.GetRecentFiles()
}

// diffExamples is the hardcoded catalog of embedded diff examples.
var diffExamples = []DiffExample{
	{Name: "add-column", Title: "Add Column", Description: "Add varchar NOT NULL column with default"},
	{Name: "add-table", Title: "Add Table", Description: "Create table + index + 2 FK"},
	{Name: "move-column", Title: "Move Column", Description: "Drop column from one table, add to another (DELETES_DATA)"},
	{Name: "modify-index", Title: "Modify Index", Description: "Change index columns + add WHERE predicate"},
}

// ListDiffExamples returns available pre-built diff examples.
//
//zenrpc:return []DiffExample
func (s *AppService) ListDiffExamples() []DiffExample {
	return diffExamples
}

// RunDiffExample loads a diff pair and returns the diff result.
//
//zenrpc:name diff example name (add-column, add-table, move-column, modify-index)
//zenrpc:return DiffUnsavedResult
func (s *AppService) RunDiffExample(name string) (*DiffUnsavedResult, error) {
	oldProject, err := loadDiffExample(name, "old.pgd")
	if err != nil {
		return nil, fmt.Errorf("loading old: %w", err)
	}
	newProject, err := loadDiffExample(name, "new.pgd")
	if err != nil {
		return nil, fmt.Errorf("loading new: %w", err)
	}
	result := diff.Diff(oldProject, newProject)
	return &DiffUnsavedResult{
		SQL:     result.SQL(),
		Changes: NewDiffChanges(result.Changes),
	}, nil
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

// pgdFilePath returns the .pgd output path for a given input.
func pgdFilePath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".pgd":
		return path
	case ".pdd", ".dbs", ".dm2", ".sql":
		return strings.TrimSuffix(path, ext) + ".pgd"
	}
	// DSN or unknown format — no file path (requires Save As)
	return ""
}

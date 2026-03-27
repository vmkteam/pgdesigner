package rpc

import (
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/vmkteam/pgdesigner/pkg/designer"
	"github.com/vmkteam/pgdesigner/pkg/designer/store"
	"github.com/vmkteam/zenrpc/v2"
)

const quitGracePeriod = 3 * time.Second

// ConfigCallbacks provides access to app config without circular imports.
type ConfigCallbacks struct {
	Register         func(email string) error
	IsRegistered     func() bool
	GetRecentFiles   func() []string
	AddRecentFile    func(path string) error
	RemoveRecentFile func(path string) error
	CheckUpdate      func(currentVersion string) UpdateInfo
	DismissVersion   func(version string) error
}

// AppService provides application lifecycle methods.
type AppService struct {
	zenrpc.Service
	mgr     *designer.AppManager
	store   *store.ProjectStore
	config  ConfigCallbacks
	quitCh  chan struct{}
	version string
	mu      sync.Mutex //nolint:govet // zenrpc generates value-receiver methods; mutex is safe because AppService is always used via pointer
	timer   *time.Timer
}

// NewAppService creates an AppService that signals quit via the provided channel.
func NewAppService(quitCh chan struct{}, s *store.ProjectStore, cfg ConfigCallbacks, version string) *AppService {
	return &AppService{
		mgr:     designer.NewAppManager(),
		store:   s,
		config:  cfg,
		quitCh:  quitCh,
		version: version,
	}
}

// Quit starts a delayed shutdown. If Ping is not called within the grace period, the server exits.
//
// zenrpc
func (s *AppService) Quit() {
	s.mu.Lock()
	defer s.mu.Unlock()

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

// About returns application metadata.
//
//zenrpc:return AboutInfo
func (s *AppService) About() AboutInfo {
	return AboutInfo{
		Name:        "PgDesigner",
		Description: "Visual PostgreSQL Schema Designer",
		Version:     s.version,
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
	return NewDemoSchemasFromInfo(s.mgr.ListDemoSchemas())
}

// OpenDemo loads an embedded demo schema by name.
//
//zenrpc:name demo schema name (chinook, northwind, pagila, airlines, adventureworks)
//zenrpc:return bool
func (s *AppService) OpenDemo(name string) (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	project, err := s.mgr.OpenDemo(name)
	if err != nil {
		return false, err
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
	project, pgdPath, err := s.mgr.OpenFile(path)
	if err != nil {
		return false, err
	}
	s.store.ReplaceProject(project, pgdPath)
	if s.config.AddRecentFile != nil && pgdPath != "" {
		_ = s.config.AddRecentFile(path)
	}
	return true, nil
}

// NewProject creates a new empty project, replacing the current one.
//
//zenrpc:return bool
func (s *AppService) NewProject() (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	s.store.ReplaceProject(s.mgr.NewProject(), "")
	return true, nil
}

// CloseProject replaces current project with empty one (returns to welcome screen).
//
//zenrpc:return bool
func (s *AppService) CloseProject() (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	s.store.ReplaceProject(s.mgr.NewProject(), "")
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

// GetHomePath returns the user's home directory path.
//
//zenrpc:return string
func (s *AppService) GetHomePath() string {
	return s.mgr.GetHomePath()
}

// ListDirectory lists files and subdirectories at the given path.
// Returns entries sorted: directories first (alphabetical), then files (alphabetical).
// Hidden files (starting with .) are excluded.
//
//zenrpc:path      absolute directory path (~ expanded server-side)
//zenrpc:showAll   if true, show all files; if false, only supported extensions
//zenrpc:return    DirectoryListing
func (s *AppService) ListDirectory(path string, showAll bool) (*DirectoryListing, error) {
	dl, err := s.mgr.ListDirectory(path, showAll)
	if err != nil {
		return nil, err
	}
	return NewDirectoryListingFromDirListing(dl), nil
}

// RemoveRecentFile removes a path from the recent files list.
//
//zenrpc:path file path to remove
//zenrpc:return bool
func (s *AppService) RemoveRecentFile(path string) (bool, error) {
	if s.config.RemoveRecentFile == nil {
		return false, errors.New("config not available")
	}
	if err := s.config.RemoveRecentFile(path); err != nil {
		return false, err
	}
	return true, nil
}

// GetRecentFilesInfo returns recent files with metadata (size, mod time, exists).
//
//zenrpc:return []RecentFile
func (s *AppService) GetRecentFilesInfo() []RecentFile {
	if s.config.GetRecentFiles == nil {
		return nil
	}
	paths := s.config.GetRecentFiles()
	return NewRecentFilesFromInfo(s.mgr.GetRecentFilesInfo(paths))
}

// ListDiffExamples returns available pre-built diff examples.
//
//zenrpc:return []DiffExample
func (s *AppService) ListDiffExamples() []DiffExample {
	return NewDiffExamplesFromInfo(s.mgr.ListDiffExamples())
}

// RunDiffExample loads a diff pair and returns the diff result.
//
//zenrpc:name diff example name (add-column, add-table, move-column, modify-index)
//zenrpc:return DiffUnsavedResult
func (s *AppService) RunDiffExample(name string) (*DiffUnsavedResult, error) {
	result, err := s.mgr.RunDiffExample(name)
	if err != nil {
		return nil, err
	}
	return &DiffUnsavedResult{
		SQL:     result.SQL,
		Changes: NewDiffChanges(result.Changes),
	}, nil
}

// CheckForUpdate checks GitHub Releases for a newer version of PgDesigner.
// Results are cached for 24 hours. Safe to call in read-only mode.
//
//zenrpc:return UpdateInfo
func (s *AppService) CheckForUpdate() *UpdateInfo {
	if s.config.CheckUpdate == nil {
		return nil
	}
	result := s.config.CheckUpdate(s.version)
	return &UpdateInfo{
		CurrentVersion:  result.CurrentVersion,
		LatestVersion:   result.LatestVersion,
		UpdateAvailable: result.UpdateAvailable,
		ReleaseURL:      result.ReleaseURL,
		ShouldNotify:    result.ShouldNotify,
	}
}

// DismissUpdate records that the user has dismissed the update notification for the given version.
//
//zenrpc:version version string to dismiss (e.g. "v0.2.0")
//zenrpc:return bool
func (s *AppService) DismissUpdate(version string) (bool, error) {
	if s.config.DismissVersion == nil {
		return false, errors.New("config not available")
	}
	if err := s.config.DismissVersion(version); err != nil {
		return false, err
	}
	return true, nil
}

// IntrospectDSN connects to a PostgreSQL database and returns a preview of available objects.
//
//zenrpc:dsn PostgreSQL connection string
//zenrpc:return DSNPreview
func (s *AppService) IntrospectDSN(dsn string) (*DSNPreview, error) {
	result, err := s.mgr.IntrospectDSN(dsn)
	if err != nil {
		return nil, err
	}
	return NewDSNPreview(result), nil
}

// ImportDSN imports schema from PostgreSQL with filtering options.
//
//zenrpc:dsn        PostgreSQL connection string
//zenrpc:schemas    schemas to import (empty = all)
//zenrpc:tables     specific tables to import as "schema.table" (empty = all in selected schemas)
//zenrpc:categories object categories to include: views, matviews, functions, triggers, enums, domains, sequences, extensions
//zenrpc:return bool
func (s *AppService) ImportDSN(dsn string, schemas []string, tables []string, categories []string) (bool, error) {
	if s.store == nil {
		return false, errors.New("store not available")
	}
	project, err := s.mgr.ImportDSN(dsn, designer.ImportDSNOptions{
		Schemas:    schemas,
		Tables:     tables,
		Categories: categories,
	})
	if err != nil {
		return false, err
	}
	s.store.ReplaceProject(project, "")
	return true, nil
}

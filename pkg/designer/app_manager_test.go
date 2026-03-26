package designer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppManager_NewProject(t *testing.T) {
	mgr := NewAppManager()
	p := mgr.NewProject()
	require.NotNil(t, p)
	assert.Equal(t, "18", p.PgVersion)
}

func TestAppManager_ListDemoSchemas(t *testing.T) {
	mgr := NewAppManager()
	schemas := mgr.ListDemoSchemas()
	assert.Len(t, schemas, 5)
	assert.Equal(t, "chinook", schemas[0].Name)
	assert.Equal(t, "adventureworks", schemas[4].Name)
}

func TestAppManager_OpenDemo(t *testing.T) {
	mgr := NewAppManager()

	t.Run("valid", func(t *testing.T) {
		p, err := mgr.OpenDemo("chinook")
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.NotEmpty(t, p.Schemas)
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := mgr.OpenDemo("nonexistent")
		assert.Error(t, err)
	})
}

func TestAppManager_OpenFile(t *testing.T) {
	mgr := NewAppManager()

	t.Run("nonexistent", func(t *testing.T) {
		_, _, err := mgr.OpenFile("/tmp/does-not-exist.pgd")
		assert.Error(t, err)
	})
}

func TestAppManager_ListDiffExamples(t *testing.T) {
	mgr := NewAppManager()
	examples := mgr.ListDiffExamples()
	assert.Len(t, examples, 4)
	assert.Equal(t, "add-column", examples[0].Name)
}

func TestAppManager_RunDiffExample(t *testing.T) {
	mgr := NewAppManager()

	t.Run("valid", func(t *testing.T) {
		result, err := mgr.RunDiffExample("add-column")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.SQL)
		assert.NotEmpty(t, result.Changes)
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := mgr.RunDiffExample("nonexistent")
		assert.Error(t, err)
	})
}

func TestAppManager_GetHomePath(t *testing.T) {
	mgr := NewAppManager()
	home := mgr.GetHomePath()
	assert.NotEmpty(t, home)
	assert.True(t, filepath.IsAbs(home))
}

func TestAppManager_ListDirectory(t *testing.T) {
	mgr := NewAppManager()

	t.Run("valid directory", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "schema.pgd"), []byte("<project/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0o644))

		// filtered mode
		dl, err := mgr.ListDirectory(dir, false)
		require.NoError(t, err)
		assert.Equal(t, dir, dl.Path)
		// subdir + schema.pgd (notes.txt filtered, .hidden skipped)
		assert.Len(t, dl.Entries, 2)
		assert.True(t, dl.Entries[0].IsDir)
		assert.Equal(t, "subdir", dl.Entries[0].Name)
		assert.Equal(t, "schema.pgd", dl.Entries[1].Name)
		assert.True(t, dl.Entries[1].Supported)

		// show all mode
		dl, err = mgr.ListDirectory(dir, true)
		require.NoError(t, err)
		// subdir + notes.txt + schema.pgd (.hidden still skipped)
		assert.Len(t, dl.Entries, 3)
		assert.Equal(t, "subdir", dl.Entries[0].Name)
		assert.Equal(t, "notes.txt", dl.Entries[1].Name)
		assert.False(t, dl.Entries[1].Supported)
		assert.Equal(t, "schema.pgd", dl.Entries[2].Name)
	})

	t.Run("nonexistent", func(t *testing.T) {
		_, err := mgr.ListDirectory("/tmp/nonexistent-dir-12345", false)
		assert.Error(t, err)
	})

	t.Run("blocked dir", func(t *testing.T) {
		dl, err := mgr.ListDirectory("/proc", false)
		require.NoError(t, err)
		assert.Empty(t, dl.Entries)
	})

	t.Run("tilde expansion", func(t *testing.T) {
		dl, err := mgr.ListDirectory("~", false)
		require.NoError(t, err)
		home, _ := os.UserHomeDir()
		assert.Equal(t, home, dl.Path)
	})

	t.Run("dirs first then files sorted", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "beta"), 0o755))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "alpha"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.pgd"), []byte("<project/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.pgd"), []byte("<project/>"), 0o644))

		dl, err := mgr.ListDirectory(dir, false)
		require.NoError(t, err)
		require.Len(t, dl.Entries, 4)
		assert.Equal(t, "alpha", dl.Entries[0].Name)
		assert.Equal(t, "beta", dl.Entries[1].Name)
		assert.Equal(t, "a.pgd", dl.Entries[2].Name)
		assert.Equal(t, "b.pgd", dl.Entries[3].Name)
	})
}

func TestAppManager_GetRecentFilesInfo(t *testing.T) {
	mgr := NewAppManager()

	t.Run("existing file", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "test.pgd")
		require.NoError(t, os.WriteFile(f, []byte("data"), 0o644))

		infos := mgr.GetRecentFilesInfo([]string{f})
		require.Len(t, infos, 1)
		assert.True(t, infos[0].Exists)
		assert.Equal(t, "test.pgd", infos[0].Name)
		assert.Equal(t, int64(4), infos[0].Size)
		assert.False(t, infos[0].ModTime.IsZero())
	})

	t.Run("missing file", func(t *testing.T) {
		infos := mgr.GetRecentFilesInfo([]string{"/tmp/gone-12345.pgd"})
		require.Len(t, infos, 1)
		assert.False(t, infos[0].Exists)
		assert.Equal(t, "gone-12345.pgd", infos[0].Name)
		assert.Equal(t, int64(0), infos[0].Size)
	})

	t.Run("empty list", func(t *testing.T) {
		infos := mgr.GetRecentFilesInfo(nil)
		assert.Empty(t, infos)
	})
}

func Test_pgdFilePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/path/to/schema.pgd", "/path/to/schema.pgd"},
		{"/path/to/schema.pdd", "/path/to/schema.pgd"},
		{"/path/to/schema.dbs", "/path/to/schema.pgd"},
		{"/path/to/schema.dm2", "/path/to/schema.pgd"},
		{"/path/to/dump.sql", "/path/to/dump.pgd"},
		{"postgres://user:pass@localhost/db", ""},
		{"/path/to/unknown.xyz", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, pgdFilePath(tt.input))
		})
	}
}

func Test_expandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~", home},
		{"~/Documents", filepath.Join(home, "Documents")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, expandHome(tt.input))
		})
	}
}

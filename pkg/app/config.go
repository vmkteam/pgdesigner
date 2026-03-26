package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	appDir     = "pgdesigner"
	configFile = "config.json"
	maxRecent  = 10
)

// Config holds application-level settings stored in ~/.config/pgdesigner/config.json.
type Config struct {
	RegisteredEmail string   `json:"registeredEmail,omitempty"`
	RecentFiles     []string `json:"recentFiles,omitempty"`
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDir, configFile), nil
}

// Load reads the config from os.UserConfigDir()/pgdesigner/config.json.
// If the file does not exist, it returns an empty Config without error.
func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save writes the config to os.UserConfigDir()/pgdesigner/config.json,
// creating the directory if needed.
func (c *Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}

	if mkErr := os.MkdirAll(filepath.Dir(p), 0o755); mkErr != nil {
		return mkErr
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0o644)
}

// IsRegistered returns true when a registered email is set.
func (c *Config) IsRegistered() bool {
	return c.RegisteredEmail != ""
}

// AddRecentFile prepends path to the recent files list, removes duplicates,
// and keeps at most 10 entries.
func (c *Config) AddRecentFile(path string) {
	files := make([]string, 0, maxRecent)
	files = append(files, path)
	for _, f := range c.RecentFiles {
		if f == path {
			continue
		}
		files = append(files, f)
		if len(files) == maxRecent {
			break
		}
	}
	c.RecentFiles = files
}

// RemoveRecentFile removes a path from the recent files list.
func (c *Config) RemoveRecentFile(path string) {
	files := make([]string, 0, len(c.RecentFiles))
	for _, f := range c.RecentFiles {
		if f != path {
			files = append(files, f)
		}
	}
	c.RecentFiles = files
}

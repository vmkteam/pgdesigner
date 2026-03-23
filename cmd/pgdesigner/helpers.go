package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/format"
	"github.com/vmkteam/pgdesigner/pkg/format/dbs"
	"github.com/vmkteam/pgdesigner/pkg/format/dm2"
	"github.com/vmkteam/pgdesigner/pkg/format/pdd"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	sqlfmt "github.com/vmkteam/pgdesigner/pkg/format/sql"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

var converters = map[string]format.Converter{
	format.ExtDBS: format.ConverterFunc(dbs.Convert),
	format.ExtDM2: format.ConverterFunc(dm2.Convert),
	format.ExtPDD: format.ConverterFunc(pdd.Convert),
	format.ExtSQL: format.ConverterFunc(sqlfmt.Convert),
}

// loadFile loads a .pgd file, auto-converts other formats, or introspects a PostgreSQL DSN.
func loadFile(path string, opts pgre.Options) (*pgd.Project, error) {
	if pgre.IsDSN(path) {
		return pgre.Connect(path, opts)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	if c, ok := converters[ext]; ok {
		name := strings.TrimSuffix(filepath.Base(path), ext)
		return c.Convert(data, name)
	}

	var project pgd.Project
	if err := xml.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}
	return &project, nil
}

// saveProject writes a project to a .pgd file.
func saveProject(p *pgd.Project, path string) error {
	data, err := xml.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal XML: %w", err)
	}
	return os.WriteFile(path, []byte(xml.Header+string(data)+"\n"), 0644)
}

// pgdFilePath returns the .pgd output path for a given input.
// For non-.pgd inputs it replaces the extension so we never overwrite the source file.
func pgdFilePath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".pgd" && ext != "" {
		return strings.TrimSuffix(path, filepath.Ext(path)) + ".pgd"
	}
	return path
}

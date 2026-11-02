package format

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/format/dbs"
	"github.com/vmkteam/pgdesigner/pkg/format/dm2"
	"github.com/vmkteam/pgdesigner/pkg/format/pdd"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
	sqlfmt "github.com/vmkteam/pgdesigner/pkg/format/sql"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// converters maps file extensions to their format converters.
var converters = map[string]Converter{
	ExtDBS: ConverterFunc(dbs.Convert),
	ExtDM2: ConverterFunc(dm2.Convert),
	ExtPDD: ConverterFunc(pdd.Convert),
	ExtSQL: ConverterFunc(sqlfmt.Convert),
}

// options holds configuration for LoadFile.
type options struct {
	pgre pgre.Options
}

// Option configures LoadFile behavior.
type Option func(*options)

// WithSchemas sets the schemas to introspect when loading from a PostgreSQL DSN.
func WithSchemas(schemas ...string) Option {
	return func(o *options) {
		o.pgre.Schemas = schemas
	}
}

// WithFull enables full introspection (views, functions, triggers, extensions, domains, enums)
// when loading from a PostgreSQL DSN.
func WithFull(full bool) Option {
	return func(o *options) {
		o.pgre.Full = full
	}
}

// LoadFile loads a pgd.Project from a file path or PostgreSQL DSN.
// It auto-detects the format by extension (.pgd, .dbs, .dm2, .pdd, .sql)
// or connects to a live database if the path is a PostgreSQL DSN.
// Errors name the path, with the password of a DSN hidden.
func LoadFile(path string, opts ...Option) (*pgd.Project, error) {
	project, err := loadFile(path, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", pgre.RedactDSN(path), err)
	}
	return project, nil
}

func loadFile(path string, opts ...Option) (*pgd.Project, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	if pgre.IsDSN(path) {
		return pgre.Connect(path, o.pgre)
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

// Package format defines the Converter interface for importing external schema formats.
package format

import "github.com/vmkteam/pgdesigner/pkg/pgd"

// File extensions for supported formats.
const (
	ExtPGD = ".pgd"
	ExtDBS = ".dbs"
	ExtDM2 = ".dm2"
	ExtPDD = ".pdd"
	ExtSQL = ".sql"
)

// Converter converts external schema data into a pgd.Project.
type Converter interface {
	Convert(data []byte, name string) (*pgd.Project, error)
}

// ConverterFunc is an adapter to use ordinary functions as Converter.
type ConverterFunc func(data []byte, name string) (*pgd.Project, error)

func (f ConverterFunc) Convert(data []byte, name string) (*pgd.Project, error) {
	return f(data, name)
}

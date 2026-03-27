package rpc

import (
	"time"

	"github.com/vmkteam/pgdesigner/pkg/designer"
	"github.com/vmkteam/pgdesigner/pkg/format/pgre"
)

// NewDemoSchemaFromInfo converts domain DemoSchemaInfo to RPC DemoSchema.
func NewDemoSchemaFromInfo(d designer.DemoSchemaInfo) DemoSchema {
	return DemoSchema{
		Name:   d.Name,
		Title:  d.Title,
		Tables: d.Tables,
		FKs:    d.FKs,
	}
}

// NewDemoSchemasFromInfo converts a slice of domain DemoSchemaInfo to RPC DemoSchema.
func NewDemoSchemasFromInfo(ds []designer.DemoSchemaInfo) []DemoSchema {
	out := make([]DemoSchema, len(ds))
	for i, d := range ds {
		out[i] = NewDemoSchemaFromInfo(d)
	}
	return out
}

// NewDiffExampleFromInfo converts domain DiffExampleInfo to RPC DiffExample.
func NewDiffExampleFromInfo(d designer.DiffExampleInfo) DiffExample {
	return DiffExample{
		Name:        d.Name,
		Title:       d.Title,
		Description: d.Description,
	}
}

// NewDiffExamplesFromInfo converts a slice of domain DiffExampleInfo to RPC DiffExample.
func NewDiffExamplesFromInfo(ds []designer.DiffExampleInfo) []DiffExample {
	out := make([]DiffExample, len(ds))
	for i, d := range ds {
		out[i] = NewDiffExampleFromInfo(d)
	}
	return out
}

// NewDirEntryFromInfo converts domain DirEntryInfo to RPC DirEntry.
func NewDirEntryFromInfo(e designer.DirEntryInfo) DirEntry {
	return DirEntry{
		Name:      e.Name,
		IsDir:     e.IsDir,
		Size:      e.Size,
		ModTime:   e.ModTime.Format(time.RFC3339),
		Supported: e.Supported,
	}
}

// NewDirectoryListingFromDirListing converts domain DirListing to RPC DirectoryListing.
func NewDirectoryListingFromDirListing(dl *designer.DirListing) *DirectoryListing {
	entries := make([]DirEntry, len(dl.Entries))
	for i, e := range dl.Entries {
		entries[i] = NewDirEntryFromInfo(e)
	}
	return &DirectoryListing{Path: dl.Path, Entries: entries}
}

// NewRecentFileFromInfo converts domain RecentFileInfo to RPC RecentFile.
func NewRecentFileFromInfo(r designer.RecentFileInfo) RecentFile {
	rf := RecentFile{
		Path:   r.Path,
		Name:   r.Name,
		Exists: r.Exists,
	}
	if r.Exists {
		rf.Size = r.Size
		rf.ModTime = r.ModTime.Format(time.RFC3339)
	} else {
		rf.Size = -1
	}
	return rf
}

// NewRecentFilesFromInfo converts a slice of domain RecentFileInfo to RPC RecentFile.
func NewRecentFilesFromInfo(rs []designer.RecentFileInfo) []RecentFile {
	out := make([]RecentFile, len(rs))
	for i, r := range rs {
		out[i] = NewRecentFileFromInfo(r)
	}
	return out
}

// NewDSNPreview converts pgre.PreviewResult to RPC DSNPreview.
func NewDSNPreview(p *pgre.PreviewResult) *DSNPreview {
	schemas := make([]DSNSchemaPreview, len(p.Schemas))
	for i, s := range p.Schemas {
		tables := make([]DSNTablePreview, len(s.Tables))
		for j, t := range s.Tables {
			tables[j] = DSNTablePreview{
				Name: t.Name, Columns: t.Columns, Indexes: t.Indexes,
				FKs: t.FKs, Partitioned: t.Partitioned,
			}
		}
		schemas[i] = DSNSchemaPreview{Name: s.Name, Tables: tables}
	}

	roles := make([]DSNRolePreview, len(p.Roles))
	for i, r := range p.Roles {
		roles[i] = DSNRolePreview{Name: r.Name, Login: r.Login, Members: r.Members}
	}

	return &DSNPreview{
		Database:          p.Database,
		PgVersion:         p.PgVersion,
		Schemas:           schemas,
		Views:             newDSNObjects(p.Views),
		MatViews:          newDSNObjects(p.MatViews),
		Functions:         newDSNObjects(p.Functions),
		Triggers:          newDSNObjects(p.Triggers),
		Enums:             newDSNObjects(p.Enums),
		Domains:           newDSNObjects(p.Domains),
		Sequences:         newDSNObjects(p.Sequences),
		Extensions:        newDSNObjects(p.Extensions),
		Roles:             roles,
		Grants:            p.Grants,
		DefaultPrivileges: p.DefaultPrivileges,
	}
}

func newDSNObjects(objs []pgre.ObjectPreview) []DSNObjectPreview {
	out := make([]DSNObjectPreview, len(objs))
	for i, o := range objs {
		out[i] = DSNObjectPreview{Name: o.Name, Schema: o.Schema}
	}
	return out
}

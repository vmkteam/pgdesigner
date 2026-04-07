package rpc

// ValidationErrorData is the data payload for validation errors.
type ValidationErrorData struct {
	Issues []LintIssue `json:"issues"`
}

// DiffChange represents a single schema change for diff preview.
type DiffChange struct {
	Object  string       `json:"object"`  // table, column, index, fk, pk, unique, check, enum
	Action  string       `json:"action"`  // add, drop, alter
	Table   string       `json:"table"`   // parent table (for column/constraint changes)
	Name    string       `json:"name"`    // object name
	SQL     string       `json:"sql"`     // generated ALTER/CREATE/DROP
	Hazards []DiffHazard `json:"hazards"` // warnings
}

// DiffHazard is a migration risk warning.
type DiffHazard struct {
	Level   string `json:"level"` // dangerous, warning, info
	Code    string `json:"code"`  // DELETES_DATA, TABLE_REWRITE, etc.
	Message string `json:"message"`
}

// DiffUnsavedResult holds the full diff of unsaved changes.
type DiffUnsavedResult struct {
	Changes []DiffChange `json:"changes"`
	SQL     string       `json:"sql"` // full ALTER script
}

// AboutInfo holds application metadata.
type AboutInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	GoVersion   string `json:"goVersion"`
	Target      string `json:"target"`
	Author      string `json:"author"`
	License     string `json:"license"`
	Website     string `json:"website"`
	GitHub      string `json:"github"`
}

// ProjectInfo holds project metadata.
type ProjectInfo struct {
	Name            string   `json:"name"`
	PgVersion       string   `json:"pgVersion"`
	Tables          int      `json:"tables"`
	References      int      `json:"references"`
	Indexes         int      `json:"indexes"`
	AutoSave        bool     `json:"autoSave"`
	Schemas         []string `json:"schemas"`
	DefaultNullable bool     `json:"defaultNullable"`
	IsDemo          bool     `json:"isDemo"`
	IsReadOnly      bool     `json:"isReadOnly"`
	IsRegistered    bool     `json:"isRegistered"`
	FilePath        string   `json:"filePath"`
	WorkDir         string   `json:"workDir"`
}

// DemoSchema describes an available embedded demo schema.
type DemoSchema struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Tables int    `json:"tables"`
	FKs    int    `json:"fks"`
}

// DSNPreview holds a lightweight catalog of database objects.
type DSNPreview struct {
	Database          string             `json:"database"`
	PgVersion         string             `json:"pgVersion"`
	Schemas           []DSNSchemaPreview `json:"schemas"`
	Views             []DSNObjectPreview `json:"views"`
	MatViews          []DSNObjectPreview `json:"matViews"`
	Functions         []DSNObjectPreview `json:"functions"`
	Triggers          []DSNObjectPreview `json:"triggers"`
	Enums             []DSNObjectPreview `json:"enums"`
	Domains           []DSNObjectPreview `json:"domains"`
	Sequences         []DSNObjectPreview `json:"sequences"`
	Extensions        []DSNObjectPreview `json:"extensions"`
	Roles             []DSNRolePreview   `json:"roles"`
	Grants            int                `json:"grants"`
	DefaultPrivileges int                `json:"defaultPrivileges"`
}

// DSNSchemaPreview holds schema name and table summaries.
type DSNSchemaPreview struct {
	Name   string            `json:"name"`
	Tables []DSNTablePreview `json:"tables"`
}

// DSNTablePreview holds lightweight table metadata.
type DSNTablePreview struct {
	Name        string `json:"name"`
	Columns     int    `json:"columns"`
	Indexes     int    `json:"indexes"`
	FKs         int    `json:"fks"`
	Partitioned bool   `json:"partitioned"`
}

// DSNObjectPreview holds name and schema for a database object.
type DSNObjectPreview struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

// DSNRolePreview holds lightweight role metadata.
type DSNRolePreview struct {
	Name    string `json:"name"`
	Login   bool   `json:"login"`
	Members int    `json:"members"`
}

// DiffExample describes an available pre-built diff example.
type DiffExample struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ProjectSettings holds editable project-level settings.
type ProjectSettings struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	PgVersion     string `json:"pgVersion"`
	DefaultSchema string `json:"defaultSchema"`
	// Naming
	NamingConvention string `json:"namingConvention"`
	NamingTables     string `json:"namingTables"`
	// Defaults
	DefaultNullable string `json:"defaultNullable"`
	DefaultOnDelete string `json:"defaultOnDelete"`
	DefaultOnUpdate string `json:"defaultOnUpdate"`
	// Lint
	LintIgnoreRules string `json:"lintIgnoreRules"`
	// Export
	AutoSaveDDL string `json:"autoSaveDDL"`
}

// LayoutPosition holds a table position for layout save.
type LayoutPosition struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// LintIssue represents a single validation issue.
type LintIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Title    string `json:"title"`
	Path     string `json:"path"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
}

// IgnoredRule describes one ignored lint rule entry.
type IgnoredRule struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Scope string `json:"scope"` // "project" or table name
}

// LintFixRequest identifies an issue to fix.
type LintFixRequest struct {
	Code string `json:"code"`
	Path string `json:"path"`
}

// FixLintResult contains fix results and re-validated issues.
type FixLintResult struct {
	Fixed  int         `json:"fixed"`
	Issues []LintIssue `json:"issues"`
}

// TableDetail holds full table data for the Table Editor dialog.
type TableDetail struct {
	Name        string          `json:"name"`
	Schema      string          `json:"schema"`
	Unlogged    bool            `json:"unlogged,omitempty"`
	Tablespace  string          `json:"tablespace,omitempty"`
	Comment     string          `json:"comment,omitempty"`
	Columns     []ColumnDetail  `json:"columns"`
	PK          *PKDetail       `json:"pk,omitempty"`
	Uniques     []UniqueDetail  `json:"uniques"`
	Checks      []CheckDetail   `json:"checks"`
	Excludes    []ExcludeDetail `json:"excludes"`
	FKs         []FKDetail      `json:"fks"`
	Indexes     []IndexDetail   `json:"indexes"`
	PartitionBy *PartitionByRPC `json:"partitionBy,omitempty"`
	Partitions  []PartitionRPC  `json:"partitions"`
	DDL         string          `json:"ddl"`
}

// PartitionByRPC holds partition strategy for the table editor.
type PartitionByRPC struct {
	Type    string   `json:"type"` // range | list | hash
	Columns []string `json:"columns"`
}

// PartitionRPC holds a child partition for the table editor.
type PartitionRPC struct {
	Name  string `json:"name"`
	Bound string `json:"bound"`
}

// ColumnDetail holds column data for the editor.
type ColumnDetail struct {
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Length          int             `json:"length,omitempty"`
	Precision       int             `json:"precision,omitempty"`
	Scale           int             `json:"scale,omitempty"`
	Nullable        bool            `json:"nullable"`
	Default         string          `json:"default,omitempty"`
	PK              bool            `json:"pk,omitempty"`
	FK              bool            `json:"fk,omitempty"`
	Identity        string          `json:"identity,omitempty"`
	IdentitySeqOpt  *IdentitySeqOpt `json:"identitySeqOpt,omitempty"`
	Generated       string          `json:"generated,omitempty"`
	GeneratedStored bool            `json:"generatedStored"`
	Comment         string          `json:"comment,omitempty"`
	Compression     string          `json:"compression,omitempty"`
	Storage         string          `json:"storage,omitempty"`
	Collation       string          `json:"collation,omitempty"`
}

// IdentitySeqOpt holds identity column sequence parameters.
type IdentitySeqOpt struct {
	Start     int64 `json:"start,omitempty"`
	Increment int64 `json:"increment,omitempty"`
	Min       int64 `json:"min,omitempty"`
	Max       int64 `json:"max,omitempty"`
	Cache     int64 `json:"cache,omitempty"`
	Cycle     bool  `json:"cycle,omitempty"`
}

// PKDetail holds primary key data.
type PKDetail struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// UniqueDetail holds unique constraint data.
type UniqueDetail struct {
	Name          string   `json:"name"`
	Columns       []string `json:"columns"`
	NullsDistinct bool     `json:"nullsDistinct,omitempty"`
}

// CheckDetail holds check constraint data.
type CheckDetail struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// ExcludeDetail holds exclude constraint data.
type ExcludeDetail struct {
	Name     string                 `json:"name"`
	Using    string                 `json:"using,omitempty"`
	Elements []ExcludeElementDetail `json:"elements"`
	Where    string                 `json:"where,omitempty"`
}

// ExcludeElementDetail holds one element of an exclude constraint.
type ExcludeElementDetail struct {
	Column     string `json:"column,omitempty"`
	Expression string `json:"expression,omitempty"`
	Opclass    string `json:"opclass,omitempty"`
	With       string `json:"with"`
}

// FKDetail holds foreign key data.
type FKDetail struct {
	Name       string        `json:"name"`
	ToTable    string        `json:"toTable"`
	OnDelete   string        `json:"onDelete"`
	OnUpdate   string        `json:"onUpdate"`
	Deferrable bool          `json:"deferrable,omitempty"`
	Initially  string        `json:"initially,omitempty"`
	Columns    []FKColDetail `json:"columns"`
}

// FKColDetail holds one column mapping in a foreign key.
type FKColDetail struct {
	Name       string `json:"name"`
	References string `json:"references"`
}

// ERDSchema is the ERD schema for the frontend canvas.
type ERDSchema struct {
	Tables     []ERDTable     `json:"tables"`
	References []ERDReference `json:"references"`
}

// ERDTable represents a table on the ERD diagram.
type ERDTable struct {
	Name           string      `json:"name"`
	Schema         string      `json:"schema,omitempty"`
	X              int         `json:"x"`
	Y              int         `json:"y"`
	Columns        []ERDColumn `json:"columns"`
	Indexes        []ERDIndex  `json:"indexes"`
	Partitioned    bool        `json:"partitioned,omitempty"`
	PartitionCount int         `json:"partitionCount,omitempty"`
}

// ERDColumn represents a column in an ERD table card.
type ERDColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	PK      bool   `json:"pk,omitempty"`
	NN      bool   `json:"nn,omitempty"`
	FK      bool   `json:"fk,omitempty"`
	Default string `json:"default,omitempty"`
}

// ERDIndex represents an index shown in the ERD table card.
type ERDIndex struct {
	Name string `json:"name"`
}

// ERDReference represents a FK relationship line.
type ERDReference struct {
	Name    string `json:"name"`
	From    string `json:"from"`
	FromCol string `json:"fromCol"`
	To      string `json:"to"`
	ToCol   string `json:"toCol"`
}

// ObjectItem represents a database object for Go-To search.
type ObjectItem struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`  // table, column, index, fk, pk, unique, check, trigger, sequence, view, function, extension, domain, enum
	Table string `json:"table"` // parent table name (for focusing on canvas)
}

// IndexDetail holds index data.
type IndexDetail struct {
	Name          string            `json:"name"`
	Unique        bool              `json:"unique,omitempty"`
	NullsDistinct bool              `json:"nullsDistinct,omitempty"`
	Using         string            `json:"using,omitempty"`
	Columns       []IndexColDetail  `json:"columns"`
	Expressions   []string          `json:"expressions,omitempty"`
	With          []WithParamDetail `json:"with,omitempty"`
	Where         string            `json:"where,omitempty"`
	Include       []string          `json:"include,omitempty"`
}

// WithParamDetail holds a key-value storage parameter.
type WithParamDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IndexColDetail holds index column with ordering metadata.
type IndexColDetail struct {
	Name    string `json:"name"`
	Order   string `json:"order,omitempty"` // asc|desc
	Nulls   string `json:"nulls,omitempty"` // first|last
	Opclass string `json:"opclass,omitempty"`
}

// DirEntry represents a file or directory in a directory listing.
type DirEntry struct {
	Name      string `json:"name"`
	IsDir     bool   `json:"isDir"`
	Size      int64  `json:"size"`
	ModTime   string `json:"modTime"`
	Supported bool   `json:"supported"`
}

// DirectoryListing holds the result of listing a directory.
type DirectoryListing struct {
	Path    string     `json:"path"`
	Entries []DirEntry `json:"entries"`
}

// RecentFile holds metadata about a recently opened file.
type RecentFile struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Exists  bool   `json:"exists"`
}

// UpdateInfo holds the result of an update check.
type UpdateInfo struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseURL"`
	ShouldNotify    bool   `json:"shouldNotify"`
}

// TypeInfo describes a type available for column autocomplete.
type TypeInfo struct {
	Name     string `json:"name"`
	Category string `json:"category"` // numeric, character, datetime, boolean, json, network, geometric, search, array, enum, composite, domain, system, other
	Source   string `json:"source"`   // builtin, user
}

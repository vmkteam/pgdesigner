// Package pgd defines the .pgd XML format model for PgDesigner.
//
// The model covers PostgreSQL 18 DDL objects: tables, columns, constraints,
// indexes, types, views, functions, triggers, policies, roles, extensions,
// grants, comments, and diagram layouts.
//
// XSD schema: docs/architecture/pgd-format.xsd
package pgd

import "encoding/xml"

// Project is the root element of a .pgd file.
type Project struct {
	XMLName       xml.Name `xml:"pgd"`
	Version       int      `xml:"version,attr"`
	PgVersion     string   `xml:"pg-version,attr"`
	DefaultSchema string   `xml:"default-schema,attr"`

	ProjectMeta ProjectMeta  `xml:"project"`
	Database    *Database    `xml:"database,omitempty"`
	Roles       []Role       `xml:"roles>role,omitempty"`
	Tablespaces []Tablespace `xml:"tablespaces>tablespace,omitempty"`
	Extensions  []Extension  `xml:"extensions>extension,omitempty"`
	Types       *Types       `xml:"types,omitempty"`
	Sequences   []Sequence   `xml:"sequences>sequence,omitempty"`
	Schemas     []Schema     `xml:"schema"`
	Views       *Views       `xml:"views,omitempty"`
	Functions   []Function   `xml:"functions>function,omitempty"`
	Triggers    []Trigger    `xml:"triggers>trigger,omitempty"`
	Rules       []Rule       `xml:"rules>rule,omitempty"`
	Policies    []Policy     `xml:"policies>policy,omitempty"`
	Comments    []Comment    `xml:"comments>comment,omitempty"`
	Grants      *Grants      `xml:"grants,omitempty"`
	Layouts     Layouts      `xml:"layouts"`
}

// ProjectMeta holds project-level settings. // pgdesigner-specific
type ProjectMeta struct {
	Name        string   `xml:"name,attr"`
	Description string   `xml:"description,attr,omitempty"`
	Settings    Settings `xml:"settings"`
}

// Settings configures naming conventions and defaults. // pgdesigner-specific
type Settings struct {
	Naming   Naming   `xml:"naming"`
	Defaults Defaults `xml:"defaults"`
	Lint     *Lint    `xml:"lint,omitempty"`
}

// Lint configures validation rules. // pgdesigner-specific
type Lint struct {
	IgnoreRules string `xml:"ignore-rules,attr,omitempty"` // comma-separated codes: "W015,I009"
}

// Naming defines identifier naming convention. // pgdesigner-specific
type Naming struct {
	Convention string `xml:"convention,attr"`       // camelCase | snake_case | PascalCase
	Tables     string `xml:"tables,attr,omitempty"` // plural | singular (empty = no check)
}

// Defaults defines project-wide defaults. // pgdesigner-specific
type Defaults struct {
	Nullable string `xml:"nullable,attr"`
	OnDelete string `xml:"on-delete,attr"`
	OnUpdate string `xml:"on-update,attr"`
}

// Role represents a PostgreSQL role.
type Role struct {
	Name              string   `xml:"name,attr"`
	Login             string   `xml:"login,attr,omitempty"`
	Inherit           string   `xml:"inherit,attr,omitempty"`
	Createdb          string   `xml:"createdb,attr,omitempty"`
	Createrole        string   `xml:"createrole,attr,omitempty"`
	Superuser         string   `xml:"superuser,attr,omitempty"`
	Replication       string   `xml:"replication,attr,omitempty"`
	Bypassrls         string   `xml:"bypassrls,attr,omitempty"`
	ConnectionLimit   int      `xml:"connection-limit,attr,omitempty"`
	PasswordEncrypted string   `xml:"password-encrypted,attr,omitempty"`
	ValidUntil        string   `xml:"valid-until,attr,omitempty"`
	InRoles           []InRole `xml:"in-role,omitempty"`
}

// InRole references a parent role.
type InRole struct {
	Name string `xml:"name,attr"`
}

// Database represents CREATE DATABASE parameters.
type Database struct {
	Name       string `xml:"name,attr"`
	Encoding   string `xml:"encoding,attr,omitempty"`   // e.g. UTF8
	Collation  string `xml:"collation,attr,omitempty"`  // LC_COLLATE
	CType      string `xml:"ctype,attr,omitempty"`      // LC_CTYPE
	ICULocale  string `xml:"icu-locale,attr,omitempty"` // PG15+ ICU
	Locale     string `xml:"locale,attr,omitempty"`
	Template   string `xml:"template,attr,omitempty"`
	Tablespace string `xml:"tablespace,attr,omitempty"`
	Owner      string `xml:"owner,attr,omitempty"`
}

// Tablespace represents CREATE TABLESPACE.
type Tablespace struct {
	Name     string `xml:"name,attr"`
	Location string `xml:"location,attr"`
	Owner    string `xml:"owner,attr,omitempty"`
}

// Extension represents CREATE EXTENSION.
type Extension struct {
	Name    string `xml:"name,attr"`
	Schema  string `xml:"schema,attr,omitempty"`
	Version string `xml:"version,attr,omitempty"`
}

// Types groups type definitions.
type Types struct {
	Enums      []Enum      `xml:"enum,omitempty"`
	Composites []Composite `xml:"composite,omitempty"`
	Domains    []Domain    `xml:"domain,omitempty"`
	Ranges     []Range     `xml:"range,omitempty"`
}

// Enum represents CREATE TYPE ... AS ENUM.
type Enum struct {
	Name   string   `xml:"name,attr"`
	Schema string   `xml:"schema,attr,omitempty"`
	Labels []string `xml:"label"`
}

// Composite represents CREATE TYPE ... AS (fields).
type Composite struct {
	Name   string           `xml:"name,attr"`
	Schema string           `xml:"schema,attr,omitempty"`
	Fields []CompositeField `xml:"field"`
}

// CompositeField is a field in a composite type.
type CompositeField struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`
	Length    int    `xml:"length,attr,omitempty"`
	Collation string `xml:"collation,attr,omitempty"`
}

// Domain represents CREATE DOMAIN.
type Domain struct {
	Name        string             `xml:"name,attr"`
	Schema      string             `xml:"schema,attr,omitempty"`
	Type        string             `xml:"type,attr"`
	Length      int                `xml:"length,attr,omitempty"`
	Default     string             `xml:"default,attr,omitempty"`
	Collation   string             `xml:"collation,attr,omitempty"`
	NotNull     *struct{}          `xml:"not-null,omitempty"`
	Constraints []DomainConstraint `xml:"constraint,omitempty"`
}

// DomainConstraint is a CHECK constraint on a domain.
type DomainConstraint struct {
	Name       string `xml:"name,attr"`
	Expression string `xml:",cdata"`
}

// Range represents CREATE TYPE ... AS RANGE.
type Range struct {
	Name    string `xml:"name,attr"`
	Schema  string `xml:"schema,attr,omitempty"`
	Subtype string `xml:"subtype,attr"`
}

// Sequence represents CREATE SEQUENCE.
type Sequence struct {
	Name      string `xml:"name,attr"`
	Schema    string `xml:"schema,attr,omitempty"`
	Type      string `xml:"type,attr,omitempty"` // AS type (PG10+): smallint | integer | bigint
	Start     int64  `xml:"start,attr,omitempty"`
	Increment int64  `xml:"increment,attr,omitempty"`
	Min       int64  `xml:"min,attr,omitempty"`
	Max       int64  `xml:"max,attr,omitempty"`
	Cache     int64  `xml:"cache,attr,omitempty"`
	Cycle     string `xml:"cycle,attr,omitempty"`
	OwnedBy   string `xml:"owned-by,attr,omitempty"` // table.column
}

// Schema groups tables and indexes within a database schema.
type Schema struct {
	Name    string  `xml:"name,attr"`
	Tables  []Table `xml:"table"`
	Indexes []Index `xml:"index,omitempty"`
}

// Table represents a PostgreSQL table.
type Table struct {
	Name                  string `xml:"name,attr"`
	Unlogged              string `xml:"unlogged,attr,omitempty"`
	Temporary             string `xml:"temporary,attr,omitempty"`
	OnCommit              string `xml:"on-commit,attr,omitempty"` // preserve-rows | delete-rows | drop
	Tablespace            string `xml:"tablespace,attr,omitempty"`
	Comment               string `xml:"comment,attr,omitempty"` // pgdesigner-specific shortcut for COMMENT ON TABLE
	RowLevelSecurity      string `xml:"row-level-security,attr,omitempty"`
	ForceRowLevelSecurity string `xml:"force-row-level-security,attr,omitempty"`
	PartitionOf           string `xml:"partition-of,attr,omitempty"`
	Inherits              string `xml:"inherits,attr,omitempty"`
	Using                 string `xml:"using,attr,omitempty"`       // table access method
	Generate              string `xml:"generate,attr,omitempty"`    // pgdesigner-specific: "false" to skip DDL generation
	LintIgnore            string `xml:"lint-ignore,attr,omitempty"` // pgdesigner-specific: comma-separated rule codes to ignore

	Columns        []Column        `xml:"column"`
	PK             *PrimaryKey     `xml:"pk,omitempty"`
	FKs            []ForeignKey    `xml:"fk,omitempty"`
	Uniques        []Unique        `xml:"unique,omitempty"`
	Checks         []Check         `xml:"check,omitempty"`
	Excludes       []Exclude       `xml:"exclude,omitempty"`
	With           *With           `xml:"with,omitempty"`
	PartitionBy    *PartitionBy    `xml:"partition-by,omitempty"`
	PartitionBound *PartitionBound `xml:"partition-bound,omitempty"`
	Partitions     []Partition     `xml:"partition,omitempty"`
}

// Column represents a table column.
type Column struct {
	Name        string     `xml:"name,attr"`
	Type        string     `xml:"type,attr"`
	Length      int        `xml:"length,attr,omitempty"`
	Precision   int        `xml:"precision,attr,omitempty"`
	Scale       int        `xml:"scale,attr,omitempty"`
	Nullable    string     `xml:"nullable,attr,omitempty"` // "false" = NOT NULL
	Default     string     `xml:"default,attr,omitempty"`
	Comment     string     `xml:"comment,attr,omitempty"`
	Compression string     `xml:"compression,attr,omitempty"`
	Storage     string     `xml:"storage,attr,omitempty"`
	Collation   string     `xml:"collation,attr,omitempty"`
	Identity    *Identity  `xml:"identity,omitempty"`
	Generated   *Generated `xml:"generated,omitempty"`
}

// Identity represents GENERATED {ALWAYS|BY DEFAULT} AS IDENTITY.
type Identity struct {
	Generated string          `xml:"generated,attr"` // always | by-default
	Sequence  *IdentitySeqOpt `xml:"sequence,omitempty"`
}

// IdentitySeqOpt holds optional sequence parameters for identity columns.
type IdentitySeqOpt struct {
	Start     int64  `xml:"start,attr,omitempty"`
	Increment int64  `xml:"increment,attr,omitempty"`
	Min       int64  `xml:"min,attr,omitempty"`
	Max       int64  `xml:"max,attr,omitempty"`
	Cache     int64  `xml:"cache,attr,omitempty"`
	Cycle     string `xml:"cycle,attr,omitempty"`
}

// Generated represents GENERATED ALWAYS AS (expression) STORED|VIRTUAL.
type Generated struct {
	Expression string `xml:"expression,attr"`
	Stored     string `xml:"stored,attr"` // "true" = STORED, "false" = VIRTUAL (PG18)
}

// PrimaryKey represents a PRIMARY KEY constraint.
type PrimaryKey struct {
	Name            string   `xml:"name,attr"`
	WithoutOverlaps string   `xml:"without-overlaps,attr,omitempty"`
	Columns         []ColRef `xml:"column"`
}

// ForeignKey represents a FOREIGN KEY constraint.
type ForeignKey struct {
	Name       string  `xml:"name,attr"`
	ToTable    string  `xml:"to-table,attr"`
	OnDelete   string  `xml:"on-delete,attr"`
	OnUpdate   string  `xml:"on-update,attr"`
	Deferrable string  `xml:"deferrable,attr,omitempty"`
	Initially  string  `xml:"initially,attr,omitempty"`
	Match      string  `xml:"match,attr,omitempty"`
	Period     string  `xml:"period,attr,omitempty"`
	NotValid   string  `xml:"not-valid,attr,omitempty"` // pgdesigner-specific: migration state for NOT VALID
	Enforced   string  `xml:"enforced,attr,omitempty"`  // PG18: "true" | "false"
	Columns    []FKCol `xml:"column"`
}

// FKCol maps a local column to a referenced column.
type FKCol struct {
	Name       string `xml:"name,attr"`
	References string `xml:"references,attr"`
}

// Unique represents a UNIQUE constraint.
type Unique struct {
	Name          string   `xml:"name,attr"`
	NullsDistinct string   `xml:"nulls-distinct,attr,omitempty"`
	Columns       []ColRef `xml:"column"`
}

// Check represents a CHECK constraint.
type Check struct {
	Name       string `xml:"name,attr"`
	NoInherit  string `xml:"no-inherit,attr,omitempty"`
	NotValid   string `xml:"not-valid,attr,omitempty"` // pgdesigner-specific: migration state for NOT VALID
	Enforced   string `xml:"enforced,attr,omitempty"`  // PG18: "true" | "false"
	Expression string `xml:",cdata"`
}

// Exclude represents an EXCLUDE constraint.
type Exclude struct {
	Name     string           `xml:"name,attr"`
	Using    string           `xml:"using,attr,omitempty"`
	Elements []ExcludeElement `xml:"element"`
}

// ExcludeElement is one element of an EXCLUDE constraint.
type ExcludeElement struct {
	Column string `xml:"column,attr"`
	With   string `xml:"with,attr"`
}

// With holds storage parameters (fillfactor, autovacuum_enabled, etc.).
type With struct {
	Params []WithParam `xml:"param,omitempty"`
}

// WithParam is a key-value storage parameter.
type WithParam struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// PartitionBy represents PARTITION BY {RANGE|LIST|HASH}.
type PartitionBy struct {
	Type    string   `xml:"type,attr"` // range | list | hash
	Columns []ColRef `xml:"column"`
}

// PartitionBound holds raw partition bound expression.
type PartitionBound struct {
	Value string `xml:",cdata"`
}

// Partition represents a child partition of a partitioned table (Variant B — nested inside parent).
type Partition struct {
	Name        string       `xml:"name,attr"`
	Bound       string       `xml:"bound"`                  // CDATA: FOR VALUES FROM (...) TO (...) | DEFAULT
	PartitionBy *PartitionBy `xml:"partition-by,omitempty"` // multi-level partitioning
	Partitions  []Partition  `xml:"partition,omitempty"`    // sub-partitions
	Tablespace  string       `xml:"tablespace,attr,omitempty"`
	With        *With        `xml:"with,omitempty"`
}

// ColRef references a column by name with optional ordering.
type ColRef struct {
	Name    string `xml:"name,attr"`
	Order   string `xml:"order,attr,omitempty"`   // asc | desc
	Nulls   string `xml:"nulls,attr,omitempty"`   // first | last
	Opclass string `xml:"opclass,attr,omitempty"` // operator class (e.g. jsonb_path_ops, text_pattern_ops)
}

// Index represents a standalone index.
type Index struct {
	Name          string       `xml:"name,attr"`
	Table         string       `xml:"table,attr"`
	Unique        string       `xml:"unique,attr,omitempty"`
	Using         string       `xml:"using,attr,omitempty"`
	NullsDistinct string       `xml:"nulls-distinct,attr,omitempty"`
	Concurrently  string       `xml:"concurrently,attr,omitempty"` // pgdesigner-specific: DDL generation hint
	Tablespace    string       `xml:"tablespace,attr,omitempty"`
	Columns       []ColRef     `xml:"column,omitempty"`
	Expressions   []Expression `xml:"expression,omitempty"`
	Include       *Include     `xml:"include,omitempty"`
	Where         *WhereClause `xml:"where,omitempty"`
}

// Expression holds a raw SQL expression (for expression indexes).
type Expression struct {
	Value string `xml:",cdata"`
}

// Include holds INCLUDE columns for covering indexes.
type Include struct {
	Columns []ColRef `xml:"column"`
}

// WhereClause holds a partial index predicate.
type WhereClause struct {
	Value string `xml:",cdata"`
}

// Views groups view definitions.
type Views struct {
	Views    []View             `xml:"view,omitempty"`
	MatViews []MaterializedView `xml:"materialized-view,omitempty"`
}

// View represents CREATE VIEW.
type View struct {
	Name            string `xml:"name,attr"`
	Schema          string `xml:"schema,attr,omitempty"`
	Temporary       string `xml:"temporary,attr,omitempty"`
	Recursive       string `xml:"recursive,attr,omitempty"`
	SecurityInvoker string `xml:"security-invoker,attr,omitempty"`
	SecurityBarrier string `xml:"security-barrier,attr,omitempty"`
	CheckOption     string `xml:"check-option,attr,omitempty"` // local | cascaded
	Query           string `xml:"query"`
}

// MaterializedView represents CREATE MATERIALIZED VIEW.
type MaterializedView struct {
	Name       string `xml:"name,attr"`
	Schema     string `xml:"schema,attr,omitempty"`
	Tablespace string `xml:"tablespace,attr,omitempty"`
	Using      string `xml:"using,attr,omitempty"` // access method
	WithData   string `xml:"with-data,attr,omitempty"`
	Query      string `xml:"query"`
}

// Function represents CREATE FUNCTION or CREATE PROCEDURE.
type Function struct {
	Name       string    `xml:"name,attr"`
	Schema     string    `xml:"schema,attr,omitempty"`
	Kind       string    `xml:"kind,attr,omitempty"` // function (default) | procedure | aggregate
	Returns    string    `xml:"returns,attr,omitempty"`
	Language   string    `xml:"language,attr"`
	Volatility string    `xml:"volatility,attr,omitempty"` // immutable | stable | volatile
	Security   string    `xml:"security,attr,omitempty"`   // invoker | definer
	Parallel   string    `xml:"parallel,attr,omitempty"`   // unsafe | restricted | safe
	Strict     string    `xml:"strict,attr,omitempty"`     // "true" = RETURNS NULL ON NULL INPUT
	Leakproof  string    `xml:"leakproof,attr,omitempty"`
	Window     string    `xml:"window,attr,omitempty"`
	Cost       int       `xml:"cost,attr,omitempty"`
	Rows       int       `xml:"rows,attr,omitempty"`
	Args       []FuncArg `xml:"arg,omitempty"`
	RetTable   *RetTable `xml:"returns-table,omitempty"`
	Body       string    `xml:"body"`
	// Aggregate-specific fields (Kind="aggregate")
	SFunc       string `xml:"sfunc,attr,omitempty"`
	SType       string `xml:"stype,attr,omitempty"`
	FinalFunc   string `xml:"finalfunc,attr,omitempty"`
	InitCond    string `xml:"initcond,attr,omitempty"`
	SortOp      string `xml:"sortop,attr,omitempty"`
	CombineFunc string `xml:"combinefunc,attr,omitempty"`
}

// FuncArg represents a function argument.
type FuncArg struct {
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr"`
	Mode    string `xml:"mode,attr,omitempty"` // in | out | inout | variadic
	Default string `xml:"default,attr,omitempty"`
}

// RetTable represents RETURNS TABLE(...).
type RetTable struct {
	Columns []RetTableCol `xml:"column"`
}

// RetTableCol is a column in RETURNS TABLE.
type RetTableCol struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

// Trigger represents CREATE TRIGGER.
type Trigger struct {
	Name        string      `xml:"name,attr"`
	Schema      string      `xml:"schema,attr,omitempty"`
	Table       string      `xml:"table,attr"`
	Timing      string      `xml:"timing,attr"`              // before | after | instead-of
	Events      string      `xml:"events,attr"`              // insert,update,delete,truncate
	UpdateOf    string      `xml:"update-of,attr,omitempty"` // comma-separated column names for UPDATE OF
	ForEach     string      `xml:"for-each,attr"`            // row | statement
	Constraint  string      `xml:"constraint,attr,omitempty"`
	When        string      `xml:"when,attr,omitempty"`
	Referencing *TriggerRef `xml:"referencing,omitempty"`
	Execute     TriggerExec `xml:"execute"`
}

// TriggerRef holds transition table references.
type TriggerRef struct {
	NewTable string `xml:"new-table,attr,omitempty"`
	OldTable string `xml:"old-table,attr,omitempty"`
}

// TriggerExec references the trigger function.
type TriggerExec struct {
	Function string `xml:"function,attr"`
}

// Policy represents CREATE POLICY (RLS).
type Policy struct {
	Name      string      `xml:"name,attr"`
	Schema    string      `xml:"schema,attr,omitempty"`
	Table     string      `xml:"table,attr"`
	Type      string      `xml:"type,attr,omitempty"`    // permissive | restrictive
	Command   string      `xml:"command,attr,omitempty"` // all | select | insert | update | delete
	To        string      `xml:"to,attr,omitempty"`
	Using     *PolicyExpr `xml:"using,omitempty"`
	WithCheck *PolicyExpr `xml:"with-check,omitempty"`
}

// PolicyExpr holds a policy expression.
type PolicyExpr struct {
	Value string `xml:",cdata"`
}

// Rule represents CREATE RULE (deprecated pattern, low priority).
type Rule struct {
	Name    string `xml:"name,attr"`
	Schema  string `xml:"schema,attr,omitempty"`
	Table   string `xml:"table,attr"`
	Event   string `xml:"event,attr"`             // select | insert | update | delete
	Instead string `xml:"instead,attr,omitempty"` // "true" = INSTEAD
	Where   string `xml:"where,attr,omitempty"`   // condition
	Actions string `xml:"actions"`                // raw SQL (CDATA)
}

// Comment represents COMMENT ON.
type Comment struct {
	On     string `xml:"on,attr"` // table | column | index | function | schema | ...
	Schema string `xml:"schema,attr,omitempty"`
	Table  string `xml:"table,attr,omitempty"`
	Name   string `xml:"name,attr,omitempty"`
	Value  string `xml:",chardata"`
}

// Grants groups privilege definitions.
type Grants struct {
	Grants     []Grant     `xml:"grant,omitempty"`
	GrantRoles []GrantRole `xml:"grant-role,omitempty"`
}

// Grant represents a privilege grant on an object.
type Grant struct {
	On         string `xml:"on,attr"`
	Schema     string `xml:"schema,attr,omitempty"`
	Name       string `xml:"name,attr,omitempty"`
	Privileges string `xml:"privileges,attr"`
	To         string `xml:"to,attr"`
}

// GrantRole represents a role membership grant.
type GrantRole struct {
	Role        string `xml:"role,attr"`
	To          string `xml:"to,attr"`
	WithInherit string `xml:"with-inherit,attr,omitempty"`
}

// Layouts groups diagram layouts. // pgdesigner-specific
type Layouts struct {
	Layouts []Layout `xml:"layout"`
}

// Layout defines a diagram layout. // pgdesigner-specific
type Layout struct {
	Name     string         `xml:"name,attr"`
	Default  string         `xml:"default,attr,omitempty"`
	Entities []LayoutEntity `xml:"entity,omitempty"`
	Groups   []LayoutGroup  `xml:"group,omitempty"`
	Notes    []LayoutNote   `xml:"note,omitempty"`
}

// LayoutEntity positions a table on the diagram. // pgdesigner-specific
type LayoutEntity struct {
	Schema string `xml:"schema,attr"`
	Table  string `xml:"table,attr"`
	X      int    `xml:"x,attr"`
	Y      int    `xml:"y,attr"`
	Color  string `xml:"color,attr,omitempty"`
}

// LayoutGroup visually groups related tables. // pgdesigner-specific
type LayoutGroup struct {
	Name    string         `xml:"name,attr"`
	Color   string         `xml:"color,attr,omitempty"`
	Members []LayoutMember `xml:"member"`
}

// LayoutMember references a table within a group. // pgdesigner-specific
type LayoutMember struct {
	Schema string `xml:"schema,attr"`
	Table  string `xml:"table,attr"`
}

// LayoutNote is a text annotation on the diagram. // pgdesigner-specific
type LayoutNote struct {
	X     int    `xml:"x,attr"`
	Y     int    `xml:"y,attr"`
	W     int    `xml:"w,attr,omitempty"`
	H     int    `xml:"h,attr,omitempty"`
	Color string `xml:"color,attr,omitempty"`
	Text  string `xml:",chardata"`
}

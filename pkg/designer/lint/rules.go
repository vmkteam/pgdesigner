package lint

// Rule code constants.
const (
	// E — Errors (block DDL)
	RuleEmptyName          = "E001"
	RuleIdentTooLong       = "E002"
	RuleDupTable           = "E003"
	RuleDupColumn          = "E004"
	RuleDupIndexName       = "E005"
	RuleDupConstraint      = "E006"
	RulePKColNotFound      = "E007"
	RuleFKColNotFound      = "E008"
	RuleFKTableNotFound    = "E009"
	RuleFKRefColNotFound   = "E010"
	RuleIdxColNotFound     = "E011"
	RuleIdxTableNotFound   = "E012"
	RuleUniqueColNotFound  = "E013"
	RuleEmptyEnum          = "E014"
	RuleDupEnumLabel       = "E015"
	RuleEmptyComposite     = "E016"
	RuleTableNoCols        = "E017"
	RuleUnknownType        = "E018"
	RuleIncludeColNotFound = "E019"
	RuleDupSchema          = "E020"
	RuleFKNoCols           = "E021"
	RulePartColNotFound    = "E022"
	RuleTrigTableNotFound  = "E023"
	RulePolTableNotFound   = "E024"
	RuleSeqTypeInvalid     = "E025"
	RuleExclColNotFound    = "E026"
	RuleDomainTypeUnknown  = "E027"
	RuleFuncNoBody         = "E028"
	RuleFuncNoLang         = "E029"
	RuleViewNoQuery        = "E030"
	RuleMultiIdentity      = "E031"
	RulePartKeyNotInPK     = "E032" // PK/UNIQUE must include all partition key columns

	// W — Warnings (DDL valid, likely problem)
	RuleFKTypeMismatch   = "W001"
	RuleMissingFKIndex   = "W002"
	RuleCircularFK       = "W003"
	RuleNoPK             = "W004"
	RuleDupIndexCols     = "W005"
	RuleNamingViolation  = "W007"
	RulePartNoChildren   = "W008"
	RuleOrphanPartition  = "W009"
	RuleIdentityDefault  = "W010"
	RuleGeneratedDefault = "W011"
	RulePKNullable       = "W012"
	RuleUnusedSequence   = "W013"
	RuleLayoutBadRef     = "W014"
	RuleFKNoAction       = "W015"
	RuleFKToNonUnique    = "W016"
	RuleOverlapIndex     = "W017"
	RuleDupFK            = "W018"
	RuleSelfFKNotNull    = "W019"
	RuleReservedWord     = "W020"
	RuleVersionFeature   = "W021" // feature requires newer PG version
	RulePartBoundOverlap = "W022" // partition bound overlaps with sibling

	// I — Info (suggestions)
	RulePreferText     = "I001"
	RuleAvoidMoney     = "I003"
	RulePreferIdentity = "I004"
	RulePreferTSTZ     = "I005"
	RuleAvoidTimeTZ    = "I006"
	RuleAvoidRules     = "I007"
	// I008 removed (varchar(255) — not a real problem)
	RulePreferJsonb       = "I009"
	RuleTextPK            = "I012"
	RuleTooManyIndexes    = "I013"
	RuleIndexOnBool       = "I016"
	RulePKNaming          = "I017"
	RuleTablePlural       = "I018"
	RuleTooManyPartitions = "I019" // partitioned table with > 100 partitions
)

// Scope defines where a rule applies.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeTable   Scope = "table"
	ScopeColumn  Scope = "column"
	ScopeIndex   Scope = "index"
	ScopeFK      Scope = "fk"
	ScopePK      Scope = "pk"
	ScopeUnique  Scope = "unique"
	ScopeCheck   Scope = "check"
	ScopeType    Scope = "type"
)

// RuleDef describes a validation rule.
type RuleDef struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Scope    Scope    `json:"scope"`
	Title    string   `json:"title"`
	Desc     string   `json:"desc"`
	Fixable  bool     `json:"fixable"`
}

func rule(code string, sev Severity, scope Scope, title, desc string) RuleDef {
	return RuleDef{Code: code, Severity: sev, Scope: scope, Title: title, Desc: desc}
}

func fixableRule(code string, sev Severity, scope Scope, title, desc string) RuleDef {
	return RuleDef{Code: code, Severity: sev, Scope: scope, Title: title, Desc: desc, Fixable: true}
}

// Rules is the registry of all validation rules.
var Rules = map[string]RuleDef{
	// Errors
	RuleEmptyName:          rule(RuleEmptyName, Error, ScopeProject, "Empty Name", "object has an empty name"),
	RuleIdentTooLong:       rule(RuleIdentTooLong, Error, ScopeProject, "Identifier Too Long", "identifier exceeds 63 chars"),
	RuleDupTable:           rule(RuleDupTable, Error, ScopeProject, "Duplicate Table", "duplicate table in schema"),
	RuleDupColumn:          rule(RuleDupColumn, Error, ScopeTable, "Duplicate Column", "duplicate column in table"),
	RuleDupIndexName:       rule(RuleDupIndexName, Error, ScopeProject, "Duplicate Index", "duplicate index name in schema"),
	RuleDupConstraint:      rule(RuleDupConstraint, Error, ScopeTable, "Duplicate Constraint", "duplicate constraint name in table"),
	RulePKColNotFound:      rule(RulePKColNotFound, Error, ScopePK, "PK Column Not Found", "PK references unknown column"),
	RuleFKColNotFound:      rule(RuleFKColNotFound, Error, ScopeFK, "FK Column Not Found", "FK references unknown local column"),
	RuleFKTableNotFound:    rule(RuleFKTableNotFound, Error, ScopeFK, "FK Table Not Found", "FK references unknown table"),
	RuleFKRefColNotFound:   rule(RuleFKRefColNotFound, Error, ScopeFK, "FK Ref Column Not Found", "FK references unknown target column"),
	RuleIdxColNotFound:     rule(RuleIdxColNotFound, Error, ScopeIndex, "Index Column Not Found", "index references unknown column"),
	RuleIdxTableNotFound:   rule(RuleIdxTableNotFound, Error, ScopeIndex, "Index Table Not Found", "index references unknown table"),
	RuleUniqueColNotFound:  rule(RuleUniqueColNotFound, Error, ScopeUnique, "Unique Column Not Found", "UNIQUE references unknown column"),
	RuleEmptyEnum:          rule(RuleEmptyEnum, Error, ScopeType, "Empty Enum", "enum has no labels"),
	RuleDupEnumLabel:       rule(RuleDupEnumLabel, Error, ScopeType, "Duplicate Enum Label", "duplicate enum label"),
	RuleEmptyComposite:     rule(RuleEmptyComposite, Error, ScopeType, "Empty Composite", "composite type has no fields"),
	RuleTableNoCols:        rule(RuleTableNoCols, Error, ScopeTable, "Table No Columns", "table has no columns"),
	RuleUnknownType:        rule(RuleUnknownType, Error, ScopeColumn, "Unknown Type", "unknown data type"),
	RuleIncludeColNotFound: rule(RuleIncludeColNotFound, Error, ScopeIndex, "Include Column Not Found", "INCLUDE references unknown column"),
	RuleDupSchema:          rule(RuleDupSchema, Error, ScopeProject, "Duplicate Schema", "duplicate schema name"),
	RuleFKNoCols:           rule(RuleFKNoCols, Error, ScopeFK, "FK No Columns", "FK has no columns"),
	RulePartColNotFound:    rule(RulePartColNotFound, Error, ScopeTable, "Partition Column Not Found", "partition key references unknown column"),
	RuleTrigTableNotFound:  rule(RuleTrigTableNotFound, Error, ScopeProject, "Trigger Table Not Found", "trigger references unknown table"),
	RulePolTableNotFound:   rule(RulePolTableNotFound, Error, ScopeProject, "Policy Table Not Found", "policy references unknown table"),
	RuleSeqTypeInvalid:     rule(RuleSeqTypeInvalid, Error, ScopeProject, "Invalid Sequence Type", "invalid sequence type"),
	RuleExclColNotFound:    rule(RuleExclColNotFound, Error, ScopeTable, "Exclude Column Not Found", "EXCLUDE references unknown column"),
	RuleDomainTypeUnknown:  rule(RuleDomainTypeUnknown, Error, ScopeType, "Domain Type Unknown", "domain base type unknown"),
	RuleFuncNoBody:         rule(RuleFuncNoBody, Error, ScopeProject, "Function No Body", "function has no body"),
	RuleFuncNoLang:         rule(RuleFuncNoLang, Error, ScopeProject, "Function No Language", "function has no language"),
	RuleViewNoQuery:        rule(RuleViewNoQuery, Error, ScopeProject, "View No Query", "view has no query"),
	RuleMultiIdentity:      rule(RuleMultiIdentity, Error, ScopeTable, "Multiple Identity Columns", "table has multiple identity columns (PG allows only one)"),
	RulePartKeyNotInPK:     rule(RulePartKeyNotInPK, Error, ScopePK, "Partition Key Not In PK", "PK/UNIQUE must include all partition key columns"),

	// Warnings
	RuleFKTypeMismatch:   rule(RuleFKTypeMismatch, Warning, ScopeFK, "FK Type Mismatch", "FK column type mismatch"),
	RuleMissingFKIndex:   fixableRule(RuleMissingFKIndex, Warning, ScopeFK, "Missing FK Index", "FK columns have no matching index"),
	RuleCircularFK:       rule(RuleCircularFK, Warning, ScopeProject, "Circular FK", "circular FK dependency"),
	RuleNoPK:             fixableRule(RuleNoPK, Warning, ScopeTable, "No Primary Key", "table has no primary key"),
	RuleDupIndexCols:     fixableRule(RuleDupIndexCols, Warning, ScopeIndex, "Duplicate Index Columns", "duplicate index columns"),
	RuleNamingViolation:  rule(RuleNamingViolation, Warning, ScopeTable, "Naming Violation", "naming convention violation"),
	RulePartNoChildren:   rule(RulePartNoChildren, Warning, ScopeTable, "Partition No Children", "partitioned table has no children"),
	RuleOrphanPartition:  rule(RuleOrphanPartition, Warning, ScopeTable, "Orphan Partition", "orphan partition"),
	RuleIdentityDefault:  fixableRule(RuleIdentityDefault, Warning, ScopeColumn, "Identity Has Default", "identity column has default"),
	RuleGeneratedDefault: fixableRule(RuleGeneratedDefault, Warning, ScopeColumn, "Generated Has Default", "generated column has default"),
	RulePKNullable:       fixableRule(RulePKNullable, Warning, ScopeColumn, "PK Column Nullable", "PK column is nullable"),
	RuleUnusedSequence:   rule(RuleUnusedSequence, Warning, ScopeProject, "Unused Sequence", "sequence has no owned-by"),
	RuleLayoutBadRef:     rule(RuleLayoutBadRef, Warning, ScopeProject, "Layout Bad Reference", "layout references unknown table"),
	RuleFKNoAction:       fixableRule(RuleFKNoAction, Warning, ScopeFK, "FK No Action", "FK uses NO ACTION default"),
	RuleFKToNonUnique:    rule(RuleFKToNonUnique, Warning, ScopeFK, "FK To Non-Unique", "FK to non-unique column"),
	RuleOverlapIndex:     fixableRule(RuleOverlapIndex, Warning, ScopeIndex, "Overlapping Index", "overlapping index (prefix)"),
	RuleDupFK:            fixableRule(RuleDupFK, Warning, ScopeFK, "Duplicate FK", "duplicate FK"),
	RuleSelfFKNotNull:    rule(RuleSelfFKNotNull, Warning, ScopeFK, "Self FK Not Null", "self-referencing FK with NOT NULL"),
	RuleReservedWord:     rule(RuleReservedWord, Warning, ScopeTable, "Reserved Word", "reserved word as identifier"),
	RuleVersionFeature:   rule(RuleVersionFeature, Warning, ScopeTable, "PG Version Feature", "feature requires newer PostgreSQL version"),
	RulePartBoundOverlap: rule(RulePartBoundOverlap, Warning, ScopeTable, "Partition Bound Overlap", "partition bound overlaps with sibling"),

	// Info
	RulePreferText:        fixableRule(RulePreferText, Info, ScopeColumn, "Prefer Text", "prefer text over char(n)"),
	RuleAvoidMoney:        fixableRule(RuleAvoidMoney, Info, ScopeColumn, "Avoid Money", "prefer numeric over money"),
	RulePreferIdentity:    fixableRule(RulePreferIdentity, Info, ScopeColumn, "Prefer Identity", "prefer identity over serial"),
	RulePreferTSTZ:        fixableRule(RulePreferTSTZ, Info, ScopeColumn, "Prefer Timestamptz", "prefer timestamptz"),
	RuleAvoidTimeTZ:       fixableRule(RuleAvoidTimeTZ, Info, ScopeColumn, "Avoid Timetz", "timetz is rarely useful"),
	RuleAvoidRules:        rule(RuleAvoidRules, Info, ScopeProject, "Avoid Rules", "prefer triggers over rules"),
	RulePreferJsonb:       fixableRule(RulePreferJsonb, Info, ScopeColumn, "Prefer JSONB", "prefer jsonb over json"),
	RuleTextPK:            rule(RuleTextPK, Info, ScopePK, "Text PK", "text/varchar PK"),
	RuleTooManyIndexes:    rule(RuleTooManyIndexes, Info, ScopeTable, "Too Many Indexes", "too many indexes (>10)"),
	RuleIndexOnBool:       rule(RuleIndexOnBool, Info, ScopeIndex, "Index On Boolean", "index on boolean column"),
	RulePKNaming:          rule(RulePKNaming, Info, ScopePK, "PK Naming", "PK column does not follow <singularTable>Id convention"),
	RuleTablePlural:       rule(RuleTablePlural, Info, ScopeTable, "Table Plural/Singular", "table name does not match plural/singular convention"),
	RuleTooManyPartitions: rule(RuleTooManyPartitions, Info, ScopeTable, "Too Many Partitions", "partitioned table has more than 100 partitions"),
}

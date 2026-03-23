package lint

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/jinzhu/inflection"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Severity classifies a validation issue.
type Severity int

const (
	Error   Severity = iota // blocks DDL generation
	Warning                 // likely problem
	Info                    // suggestion
)

func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	default:
		return "info"
	}
}

// Issue is a single validation finding.
type Issue struct {
	Severity Severity
	Code     string
	Path     string
	Message  string
}

func (i Issue) String() string {
	return fmt.Sprintf("[%s] %s: %s (%s)", i.Severity, i.Code, i.Message, i.Path)
}

// Check function types — one per iteration scope.
type (
	projectCheck func(*validator)
	tableCheck   func(*validator, string, *pgd.Table)
	indexCheck   func(*validator, string, *pgd.Index)
)

// Registries group checks by scope. Add new checks here.
var (
	projectChecks = []projectCheck{
		checkDuplicateSchemas,
		checkDuplicateTableNames,
		checkDuplicateIndexNames,
		checkDuplicateIndexCols,
		checkOverlappingIndexes,
		checkTooManyIndexes,
		checkTypes,
		checkSequences,
		checkViews,
		checkFunctions,
		checkTriggers,
		checkPolicies,
		checkLayouts,
		checkFKCycles,
		checkRules,
	}

	tableChecks = []tableCheck{
		checkTableBasic,
		checkTableNoPK,
		checkDuplicateColumns,
		checkDuplicateConstraints,
		checkPKRefs,
		checkFKRefs,
		checkDuplicateFKs,
		checkUniqueRefs,
		checkExcludeRefs,
		checkPartitionRefs,
		checkColumns,
		checkMultiIdentity,
		checkReservedWords,
		checkTableNaming,
		checkPKNaming,
		checkTablePlural,
		checkVersionFeatures,
	}

	indexChecks = []indexCheck{
		checkIndexTable,
		checkIndexColumns,
		checkIndexOnBoolean,
		checkIndexVersionFeatures,
	}
)

// validator accumulates issues while walking the project.
type validator struct {
	p      *pgd.Project
	issues []Issue

	// lookup indexes, built once
	tables      map[string]*pgd.Table             // "schema.table" → *pgd.Table
	columns     map[string]map[string]*pgd.Column // "schema.table" → colName → *pgd.Column
	indexes     map[string][]*pgd.Index           // "schema.table" → []*pgd.Index
	userTypes   map[string]bool                   // user-defined type names
	naming      string                            // convention from settings
	pgVersion   int                               // project PG target version (e.g. 18)
	ignoreRules map[string]bool                   // global codes to skip
	tableIgnore map[string]bool                   // current table's ignored codes (reset per table)
}

// Validate checks a project for errors, warnings, and suggestions.
func Validate(p *pgd.Project) []Issue {
	v := &validator{p: p}
	v.buildIndex()
	v.run()
	sort.SliceStable(v.issues, func(i, j int) bool {
		if v.issues[i].Severity != v.issues[j].Severity {
			return v.issues[i].Severity < v.issues[j].Severity
		}
		return v.issues[i].Path < v.issues[j].Path
	})
	return v.issues
}

// ValidateTable runs checks scoped to a single table and its indexes.
// If errorsOnly is true, only Error-severity issues are returned.
func ValidateTable(p *pgd.Project, qualifiedName string, errorsOnly bool) []Issue {
	v := &validator{p: p}
	v.buildIndex()

	// Find the table
	schema, table := v.findTable(qualifiedName)
	if table == nil {
		return nil
	}

	// Run table checks
	v.setTableIgnore(table)
	for _, fn := range tableChecks {
		fn(v, schema.Name, table)
	}

	// Run index checks for indexes belonging to this table
	for i := range schema.Indexes {
		idx := &schema.Indexes[i]
		if idx.Table == table.Name {
			for _, fn := range indexChecks {
				fn(v, schema.Name, idx)
			}
		}
	}

	if !errorsOnly {
		sort.SliceStable(v.issues, func(i, j int) bool {
			if v.issues[i].Severity != v.issues[j].Severity {
				return v.issues[i].Severity < v.issues[j].Severity
			}
			return v.issues[i].Path < v.issues[j].Path
		})
		return v.issues
	}

	var errs []Issue
	for _, issue := range v.issues {
		if issue.Severity == Error {
			errs = append(errs, issue)
		}
	}
	return errs
}

// findTable locates a table by qualified or plain name.
func (v *validator) findTable(qualifiedName string) (*pgd.Schema, *pgd.Table) {
	defaultSchema := v.p.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	for i := range v.p.Schemas {
		s := &v.p.Schemas[i]
		for j := range s.Tables {
			t := &s.Tables[j]
			qualName := t.Name
			if s.Name != defaultSchema {
				qualName = s.Name + "." + t.Name
			}
			if qualName == qualifiedName || t.Name == qualifiedName {
				return s, t
			}
		}
	}
	return nil, nil
}

func (v *validator) buildIndex() {
	v.tables = make(map[string]*pgd.Table)
	v.columns = make(map[string]map[string]*pgd.Column)
	v.indexes = make(map[string][]*pgd.Index)
	v.userTypes = make(map[string]bool)

	for i := range v.p.Schemas {
		s := &v.p.Schemas[i]
		for j := range s.Tables {
			t := &s.Tables[j]
			key := s.Name + "." + t.Name
			v.tables[key] = t
			cols := make(map[string]*pgd.Column, len(t.Columns))
			for k := range t.Columns {
				cols[t.Columns[k].Name] = &t.Columns[k]
			}
			v.columns[key] = cols
		}
		for j := range s.Indexes {
			idx := &s.Indexes[j]
			key := s.Name + "." + idx.Table
			v.indexes[key] = append(v.indexes[key], idx)
		}
	}

	if v.p.Types != nil {
		for _, e := range v.p.Types.Enums {
			v.addUserType(e.Schema, e.Name)
		}
		for _, c := range v.p.Types.Composites {
			v.addUserType(c.Schema, c.Name)
		}
		for _, d := range v.p.Types.Domains {
			v.addUserType(d.Schema, d.Name)
		}
		for _, r := range v.p.Types.Ranges {
			v.addUserType(r.Schema, r.Name)
		}
	}

	v.naming = v.p.ProjectMeta.Settings.Naming.Convention

	// Parse PG target version (e.g. "18" → 18, "15" → 15).
	if n, err := strconv.Atoi(v.p.PgVersion); err == nil {
		v.pgVersion = n
	}

	v.ignoreRules = make(map[string]bool)
	if v.p.ProjectMeta.Settings.Lint != nil {
		for _, code := range strings.Split(v.p.ProjectMeta.Settings.Lint.IgnoreRules, ",") {
			code = strings.TrimSpace(code)
			if code != "" {
				v.ignoreRules[code] = true
			}
		}
	}
}

func (v *validator) addUserType(schema, name string) {
	v.userTypes[strings.ToLower(name)] = true
	if schema != "" {
		v.userTypes[strings.ToLower(schema+"."+name)] = true
	}
}

func (v *validator) run() {
	for _, fn := range projectChecks {
		fn(v)
	}
	for i := range v.p.Schemas {
		s := &v.p.Schemas[i]
		for j := range s.Tables {
			v.setTableIgnore(&s.Tables[j])
			for _, fn := range tableChecks {
				fn(v, s.Name, &s.Tables[j])
			}
		}
		for j := range s.Indexes {
			for _, fn := range indexChecks {
				fn(v, s.Name, &s.Indexes[j])
			}
		}
	}
}

// --- helpers ---

func (v *validator) setTableIgnore(t *pgd.Table) {
	v.tableIgnore = make(map[string]bool)
	if t.LintIgnore != "" {
		for _, code := range strings.Split(t.LintIgnore, ",") {
			code = strings.TrimSpace(code)
			if code != "" {
				v.tableIgnore[code] = true
			}
		}
	}
}

func (v *validator) isIgnored(code string) bool {
	return v.ignoreRules[code] || v.tableIgnore[code]
}

func (v *validator) errorf(code, path, format string, args ...any) {
	if v.isIgnored(code) {
		return
	}
	v.issues = append(v.issues, Issue{Error, code, path, fmt.Sprintf(format, args...)})
}

func (v *validator) warnf(code, path, format string, args ...any) {
	if v.isIgnored(code) {
		return
	}
	v.issues = append(v.issues, Issue{Warning, code, path, fmt.Sprintf(format, args...)})
}

func (v *validator) infof(code, path, format string, args ...any) {
	if v.isIgnored(code) {
		return
	}
	v.issues = append(v.issues, Issue{Info, code, path, fmt.Sprintf(format, args...)})
}

func tpath(schema, table string) string      { return schema + "." + table }
func cpath(schema, table, col string) string { return schema + "." + table + "." + col }

// optPath returns "schema.name" if schema is non-empty, otherwise just "name".
func optPath(schema, name string) string {
	if schema != "" {
		return schema + "." + name
	}
	return name
}

func (v *validator) checkIdent(path, name string) {
	if name == "" {
		v.errorf(RuleEmptyName, path, "empty name")
	}
	if len(name) > 63 {
		v.errorf(RuleIdentTooLong, path, "identifier %q is %d chars (max 63)", name, len(name))
	}
}

func (v *validator) resolveTable(ref, defaultSchema string) (*pgd.Table, string) {
	if strings.Contains(ref, ".") {
		return v.tables[ref], ref
	}
	key := defaultSchema + "." + ref
	return v.tables[key], key
}

func (v *validator) isKnownType(typ string) bool {
	t := pgd.StripTypeParams(typ)
	return pgd.IsKnownBuiltinType(typ) || v.userTypes[t]
}

func normalizeBaseType(typ string) string {
	return pgd.NormalizeType(pgd.StripTypeParams(typ))
}

// pgReservedWords contains PostgreSQL reserved keywords that commonly clash with identifiers.
var pgReservedWords = map[string]bool{
	"all": true, "analyse": true, "analyze": true, "and": true, "any": true,
	"array": true, "as": true, "asc": true, "authorization": true,
	"between": true, "binary": true, "both": true,
	"case": true, "cast": true, "check": true, "collate": true, "column": true,
	"constraint": true, "create": true, "cross": true,
	"default": true, "deferrable": true, "desc": true, "distinct": true, "do": true,
	"else": true, "end": true, "except": true,
	"false": true, "fetch": true, "for": true, "foreign": true, "freeze": true,
	"from": true, "full": true,
	"grant": true, "group": true,
	"having": true,
	"in":     true, "initially": true, "inner": true, "intersect": true, "into": true,
	"is":      true,
	"join":    true,
	"lateral": true, "leading": true, "left": true, "like": true, "limit": true,
	"natural": true, "not": true, "null": true,
	"offset": true, "on": true, "only": true, "or": true, "order": true, "outer": true,
	"placing": true, "primary": true,
	"references": true, "returning": true, "right": true,
	"select": true, "some": true,
	"table": true, "then": true, "to": true, "trailing": true, "true": true,
	"union": true, "unique": true, "user": true, "using": true,
	"variadic": true, "verbose": true,
	"when": true, "where": true, "window": true, "with": true,
}

// --- project checks ---

func checkDuplicateSchemas(v *validator) {
	seen := make(map[string]bool, len(v.p.Schemas))
	for _, s := range v.p.Schemas {
		v.checkIdent(s.Name, s.Name)
		if seen[s.Name] {
			v.errorf(RuleDupSchema, s.Name, "duplicate schema %q", s.Name)
		}
		seen[s.Name] = true
	}
}

func checkDuplicateTableNames(v *validator) {
	for _, s := range v.p.Schemas {
		seen := make(map[string]bool, len(s.Tables))
		for _, t := range s.Tables {
			if seen[t.Name] {
				v.errorf(RuleDupTable, tpath(s.Name, t.Name), "duplicate table %q in schema %q", t.Name, s.Name)
			}
			seen[t.Name] = true
		}
	}
}

func checkDuplicateIndexNames(v *validator) {
	for _, s := range v.p.Schemas {
		seen := make(map[string]bool, len(s.Indexes))
		for _, idx := range s.Indexes {
			if seen[idx.Name] {
				v.errorf(RuleDupIndexName, tpath(s.Name, idx.Name), "duplicate index name %q", idx.Name)
			}
			seen[idx.Name] = true
		}
	}
}

func checkDuplicateIndexCols(v *validator) {
	for _, s := range v.p.Schemas {
		type sig struct{ table, cols string }
		seen := make(map[sig]string) // sig → first index name
		for _, idx := range s.Indexes {
			names := make([]string, len(idx.Columns))
			for i, c := range idx.Columns {
				names[i] = c.Name
			}
			if len(names) == 0 {
				continue
			}
			k := sig{idx.Table, strings.Join(names, ",")}
			if first, ok := seen[k]; ok {
				v.warnf(RuleDupIndexCols, tpath(s.Name, idx.Name),
					"index has same columns as %q on table %q", first, idx.Table)
			} else {
				seen[k] = idx.Name
			}
		}
	}
}

func checkTypes(v *validator) {
	if v.p.Types == nil {
		return
	}
	for _, e := range v.p.Types.Enums {
		path := optPath(e.Schema, e.Name)
		v.checkIdent(path, e.Name)
		if len(e.Labels) == 0 {
			v.errorf(RuleEmptyEnum, path, "enum has no labels")
		}
		seen := make(map[string]bool, len(e.Labels))
		for _, l := range e.Labels {
			if seen[l] {
				v.errorf(RuleDupEnumLabel, path, "duplicate enum label %q", l)
			}
			seen[l] = true
		}
	}
	for _, c := range v.p.Types.Composites {
		path := optPath(c.Schema, c.Name)
		v.checkIdent(path, c.Name)
		if len(c.Fields) == 0 {
			v.errorf(RuleEmptyComposite, path, "composite type has no fields")
		}
	}
	for _, d := range v.p.Types.Domains {
		path := optPath(d.Schema, d.Name)
		v.checkIdent(path, d.Name)
		if d.Type != "" && !pgd.IsKnownBuiltinType(d.Type) {
			v.errorf(RuleDomainTypeUnknown, path, "domain base type %q is unknown", d.Type)
		}
	}
}

func checkSequences(v *validator) {
	validTypes := map[string]bool{
		"smallint": true, "integer": true, "bigint": true, "": true,
	}
	for _, seq := range v.p.Sequences {
		v.checkIdent(seq.Name, seq.Name)
		if !validTypes[strings.ToLower(seq.Type)] {
			v.errorf(RuleSeqTypeInvalid, seq.Name, "invalid sequence type %q", seq.Type)
		}
		if seq.OwnedBy == "" {
			v.warnf(RuleUnusedSequence, seq.Name, "sequence has no owned-by")
		}
	}
}

func checkViews(v *validator) {
	if v.p.Views == nil {
		return
	}
	for _, vw := range v.p.Views.Views {
		path := optPath(vw.Schema, vw.Name)
		v.checkIdent(path, vw.Name)
		if strings.TrimSpace(vw.Query) == "" {
			v.errorf(RuleViewNoQuery, path, "view has no query")
		}
	}
	for _, mv := range v.p.Views.MatViews {
		path := optPath(mv.Schema, mv.Name)
		v.checkIdent(path, mv.Name)
		if strings.TrimSpace(mv.Query) == "" {
			v.errorf(RuleViewNoQuery, path, "materialized view has no query")
		}
	}
}

func checkFunctions(v *validator) {
	for _, f := range v.p.Functions {
		path := optPath(f.Schema, f.Name)
		v.checkIdent(path, f.Name)
		if strings.TrimSpace(f.Body) == "" {
			v.errorf(RuleFuncNoBody, path, "function has no body")
		}
		if f.Language == "" {
			v.errorf(RuleFuncNoLang, path, "function has no language")
		}
	}
}

func checkTriggers(v *validator) {
	for _, tr := range v.p.Triggers {
		path := tr.Name
		schema := tr.Schema
		if schema == "" {
			schema = v.p.DefaultSchema
		}
		if _, ok := v.tables[tpath(schema, tr.Table)]; !ok {
			v.errorf(RuleTrigTableNotFound, path, "trigger references unknown table %q", tr.Table)
		}
	}
}

func checkPolicies(v *validator) {
	for _, pol := range v.p.Policies {
		path := pol.Name
		schema := pol.Schema
		if schema == "" {
			schema = v.p.DefaultSchema
		}
		if _, ok := v.tables[tpath(schema, pol.Table)]; !ok {
			v.errorf(RulePolTableNotFound, path, "policy references unknown table %q", pol.Table)
		}
	}
}

func checkLayouts(v *validator) {
	for _, lay := range v.p.Layouts.Layouts {
		for _, ent := range lay.Entities {
			key := tpath(ent.Schema, ent.Table)
			if _, ok := v.tables[key]; !ok {
				v.warnf(RuleLayoutBadRef, lay.Name+"."+key,
					"layout references unknown table %q", key)
			}
		}
	}
}

func checkRules(v *validator) {
	if len(v.p.Rules) > 0 {
		v.infof(RuleAvoidRules, "rules", "prefer triggers over rules")
	}
}

func checkFKCycles(v *validator) {
	// Build directed graph: table → tables it references via FK.
	graph := make(map[string][]string)
	for _, s := range v.p.Schemas {
		for _, t := range s.Tables {
			from := tpath(s.Name, t.Name)
			for _, fk := range t.FKs {
				_, to := v.resolveTable(fk.ToTable, s.Name)
				if to != from {
					graph[from] = append(graph[from], to)
				}
			}
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var path []string

	var dfs func(string)
	dfs = func(node string) {
		color[node] = gray
		path = append(path, node)
		for _, next := range graph[node] {
			switch color[next] {
			case gray:
				// find cycle start in path
				for i, n := range path {
					if n == next {
						cycle := make([]string, len(path)-i+1)
						copy(cycle, path[i:])
						cycle[len(cycle)-1] = next
						v.warnf(RuleCircularFK, next, "circular FK: %s", strings.Join(cycle, " → "))
						break
					}
				}
			case white:
				dfs(next)
			}
		}
		path = path[:len(path)-1]
		color[node] = black
	}

	for node := range graph {
		if color[node] == white {
			dfs(node)
		}
	}
}

// --- table checks ---

func checkTableBasic(v *validator, schema string, t *pgd.Table) {
	path := tpath(schema, t.Name)
	v.checkIdent(path, t.Name)
	if len(t.Columns) == 0 {
		v.errorf(RuleTableNoCols, path, "table has no columns")
	}
}

func checkTableNoPK(v *validator, schema string, t *pgd.Table) {
	if t.PK == nil && t.PartitionOf == "" {
		v.warnf(RuleNoPK, tpath(schema, t.Name), "table has no primary key")
	}
}

func checkDuplicateColumns(v *validator, schema string, t *pgd.Table) {
	seen := make(map[string]bool, len(t.Columns))
	for _, c := range t.Columns {
		if seen[c.Name] {
			v.errorf(RuleDupColumn, cpath(schema, t.Name, c.Name), "duplicate column %q", c.Name)
		}
		seen[c.Name] = true
	}
}

func checkDuplicateConstraints(v *validator, schema string, t *pgd.Table) {
	seen := make(map[string]bool)
	path := tpath(schema, t.Name)
	add := func(name string) {
		if name == "" {
			return
		}
		if seen[name] {
			v.errorf(RuleDupConstraint, path, "duplicate constraint %q", name)
		}
		seen[name] = true
	}
	if t.PK != nil {
		add(t.PK.Name)
	}
	for i := range t.FKs {
		add(t.FKs[i].Name)
	}
	for i := range t.Uniques {
		add(t.Uniques[i].Name)
	}
	for i := range t.Checks {
		add(t.Checks[i].Name)
	}
	for i := range t.Excludes {
		add(t.Excludes[i].Name)
	}
}

func checkPKRefs(v *validator, schema string, t *pgd.Table) {
	if t.PK == nil {
		return
	}
	cols := v.columns[tpath(schema, t.Name)]
	for _, ref := range t.PK.Columns {
		c, ok := cols[ref.Name]
		if !ok {
			v.errorf(RulePKColNotFound, tpath(schema, t.Name), "PK references unknown column %q", ref.Name)
			continue
		}
		if c.Nullable == "true" {
			v.warnf(RulePKNullable, cpath(schema, t.Name, ref.Name), "PK column is nullable")
		}
		lt := strings.ToLower(c.Type)
		base := normalizeBaseType(lt)
		if base == "text" || base == "character varying" || base == "character" {
			v.infof(RuleTextPK, cpath(schema, t.Name, ref.Name), "PK on text/varchar column — integer is usually better")
		}
	}
}

func checkFKRefs(v *validator, schema string, t *pgd.Table) {
	path := tpath(schema, t.Name)
	cols := v.columns[path]

	for _, fk := range t.FKs {
		target, targetKey := v.resolveTable(fk.ToTable, schema)
		if target == nil {
			v.errorf(RuleFKTableNotFound, path, "FK %q references unknown table %q", fk.Name, fk.ToTable)
			continue
		}
		targetCols := v.columns[targetKey]

		if len(fk.Columns) == 0 {
			v.errorf(RuleFKNoCols, path, "FK %q has no columns", fk.Name)
			continue
		}

		for _, fc := range fk.Columns {
			local, lok := cols[fc.Name]
			if !lok {
				v.errorf(RuleFKColNotFound, path, "FK %q references unknown local column %q", fk.Name, fc.Name)
			}
			ref, rok := targetCols[fc.References]
			if !rok {
				v.errorf(RuleFKRefColNotFound, path, "FK %q references unknown column %q in %q", fk.Name, fc.References, fk.ToTable)
			}
			if lok && rok && normalizeBaseType(local.Type) != normalizeBaseType(ref.Type) {
				v.warnf(RuleFKTypeMismatch, cpath(schema, t.Name, fc.Name),
					"FK %q type mismatch: %s vs %s", fk.Name, local.Type, ref.Type)
			}
		}

		// W016: FK to non-unique columns
		v.checkFKTargetUnique(schema, t, &fk, target)

		// W002: missing index on FK columns
		v.checkFKIndex(schema, t, &fk)

		// W019: self-referencing FK with NOT NULL — can't insert first row
		if fk.ToTable == t.Name || fk.ToTable == tpath(schema, t.Name) {
			allNotNull := true
			for _, fc := range fk.Columns {
				if c, ok := cols[fc.Name]; ok && c.Nullable != "false" {
					allNotNull = false
					break
				}
			}
			if allNotNull {
				v.warnf(RuleSelfFKNotNull, path, "FK %q is self-referencing with NOT NULL columns — cannot insert first row", fk.Name)
			}
		}

		// W015: FK with NO ACTION — likely unintentional default
		del := strings.ToLower(fk.OnDelete)
		upd := strings.ToLower(fk.OnUpdate)
		if del == "no action" || upd == "no action" {
			v.warnf(RuleFKNoAction, path, "FK %q uses NO ACTION — specify action explicitly", fk.Name)
		}
	}
}

func (v *validator) checkFKIndex(schema string, t *pgd.Table, fk *pgd.ForeignKey) {
	fkCols := make([]string, len(fk.Columns))
	for i := range fk.Columns {
		fkCols[i] = fk.Columns[i].Name
	}

	// PK covers?
	if t.PK != nil && prefixMatch(t.PK.Columns, fkCols) {
		return
	}
	// Any UNIQUE covers?
	for _, u := range t.Uniques {
		if prefixMatch(u.Columns, fkCols) {
			return
		}
	}
	// Any index covers?
	for _, idx := range v.indexes[tpath(schema, t.Name)] {
		if prefixMatch(idx.Columns, fkCols) {
			return
		}
	}

	v.warnf(RuleMissingFKIndex, tpath(schema, t.Name), "FK %q columns have no matching index", fk.Name)
}

// prefixMatch reports whether refs starts with the given names.
func prefixMatch(refs []pgd.ColRef, names []string) bool {
	if len(refs) < len(names) {
		return false
	}
	for i, n := range names {
		if refs[i].Name != n {
			return false
		}
	}
	return true
}

func checkUniqueRefs(v *validator, schema string, t *pgd.Table) {
	cols := v.columns[tpath(schema, t.Name)]
	for _, u := range t.Uniques {
		for _, ref := range u.Columns {
			if _, ok := cols[ref.Name]; !ok {
				v.errorf(RuleUniqueColNotFound, tpath(schema, t.Name),
					"UNIQUE %q references unknown column %q", u.Name, ref.Name)
			}
		}
	}
}

func checkExcludeRefs(v *validator, schema string, t *pgd.Table) {
	cols := v.columns[tpath(schema, t.Name)]
	for _, ex := range t.Excludes {
		for _, el := range ex.Elements {
			if _, ok := cols[el.Column]; !ok {
				v.errorf(RuleExclColNotFound, tpath(schema, t.Name),
					"EXCLUDE %q references unknown column %q", ex.Name, el.Column)
			}
		}
	}
}

func checkPartitionRefs(v *validator, schema string, t *pgd.Table) {
	path := tpath(schema, t.Name)
	cols := v.columns[path]

	if t.PartitionBy != nil {
		for _, ref := range t.PartitionBy.Columns {
			if _, ok := cols[ref.Name]; !ok {
				v.errorf(RulePartColNotFound, path, "partition key references unknown column %q", ref.Name)
			}
		}
		if !v.hasPartitionChildren(t.Name, path, t) {
			v.warnf(RulePartNoChildren, path, "partitioned table has no child partitions")
		}
		checkPartKeyInConstraints(v, path, t)
		checkPartitionBounds(v, path, t.Partitions)
		if len(t.Partitions) > 100 {
			v.infof(RuleTooManyPartitions, path, "partitioned table has %d partitions", len(t.Partitions))
		}
	}

	// W009: orphan partition (Variant A only)
	if t.PartitionOf != "" {
		parent, _ := v.resolveTable(t.PartitionOf, schema)
		if parent != nil && parent.PartitionBy == nil {
			v.warnf(RuleOrphanPartition, path, "partition of %q but parent has no PARTITION BY", t.PartitionOf)
		}
	}
}

// hasPartitionChildren checks if table has children (Variant A or Variant B).
func (v *validator) hasPartitionChildren(name, path string, t *pgd.Table) bool {
	if len(t.Partitions) > 0 {
		return true
	}
	for _, s := range v.p.Schemas {
		for _, child := range s.Tables {
			if child.PartitionOf == name || child.PartitionOf == path {
				return true
			}
		}
	}
	return false
}

// checkPartitionBounds validates W022: partition bounds should not have duplicate/overlapping values.
func checkPartitionBounds(v *validator, path string, partitions []pgd.Partition) {
	seen := make(map[string]string) // bound → partition name
	for _, p := range partitions {
		if p.Bound == "" || p.Bound == "DEFAULT" {
			continue
		}
		if prev, ok := seen[p.Bound]; ok {
			v.warnf(RulePartBoundOverlap, path, "partition %q has same bound as %q: %s", p.Name, prev, p.Bound)
		}
		seen[p.Bound] = p.Name
	}
}

// checkPartKeyInConstraints validates E032: PK/UNIQUE must include all partition key columns.
func checkPartKeyInConstraints(v *validator, path string, t *pgd.Table) {
	if t.PK != nil {
		pkCols := make(map[string]bool)
		for _, c := range t.PK.Columns {
			pkCols[c.Name] = true
		}
		for _, ref := range t.PartitionBy.Columns {
			if !pkCols[ref.Name] {
				v.errorf(RulePartKeyNotInPK, path, "PK must include partition key column %q", ref.Name)
			}
		}
	}
	for _, u := range t.Uniques {
		uCols := make(map[string]bool)
		for _, c := range u.Columns {
			uCols[c.Name] = true
		}
		for _, ref := range t.PartitionBy.Columns {
			if !uCols[ref.Name] {
				v.errorf(RulePartKeyNotInPK, path, "UNIQUE %q must include partition key column %q", u.Name, ref.Name)
			}
		}
	}
}

func checkColumns(v *validator, schema string, t *pgd.Table) {
	for _, c := range t.Columns {
		cp := cpath(schema, t.Name, c.Name)
		v.checkIdent(cp, c.Name)

		if c.Type != "" && !v.isKnownType(c.Type) {
			v.errorf(RuleUnknownType, cp, "unknown type %q", c.Type)
		}

		if c.Identity != nil && c.Default != "" {
			v.warnf(RuleIdentityDefault, cp, "column has both identity and default")
		}
		if c.Generated != nil && c.Default != "" {
			v.warnf(RuleGeneratedDefault, cp, "column has both generated expression and default")
		}

		lt := strings.ToLower(c.Type)

		if (lt == "char" || lt == "character") && c.Length > 0 {
			v.infof(RulePreferText, cp, "prefer text over char(%d)", c.Length)
		}
		if lt == "money" {
			v.infof(RuleAvoidMoney, cp, "prefer numeric over money")
		}
		if lt == "serial" || lt == "bigserial" || lt == "smallserial" {
			v.infof(RulePreferIdentity, cp, "prefer identity column over %s", c.Type)
		}
		if lt == "timestamp" || lt == "timestamp without time zone" {
			v.infof(RulePreferTSTZ, cp, "prefer timestamptz")
		}
		if lt == "timetz" || lt == "time with time zone" {
			v.infof(RuleAvoidTimeTZ, cp, "timetz is rarely useful")
		}
		if lt == "json" {
			v.infof(RulePreferJsonb, cp, "prefer jsonb over json")
		}
	}
}

func checkTableNaming(v *validator, schema string, t *pgd.Table) {
	if v.naming == "" {
		return
	}
	path := tpath(schema, t.Name)
	v.checkNaming(path, t.Name)
	for _, c := range t.Columns {
		v.checkNaming(cpath(schema, t.Name, c.Name), c.Name)
	}
}

func (v *validator) checkNaming(path, name string) {
	// W007: naming convention
	if v.naming == "snake_case" && !isSnakeCase(name) {
		v.warnf(RuleNamingViolation, path, "%q violates snake_case convention", name)
	}
}

func isSnakeCase(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

// checkPKNaming checks that single-column PK follows <singularTable>Id / <singular_table>_id convention.
func checkPKNaming(v *validator, schema string, t *pgd.Table) {
	if t.PK == nil || len(t.PK.Columns) != 1 {
		return
	}
	pkCol := t.PK.Columns[0].Name
	expected := ExpectedPKName(t.Name, v.naming)
	if len(expected) == 0 {
		return
	}
	for _, e := range expected {
		if pkCol == e {
			return
		}
	}
	v.infof(RulePKNaming, tpath(schema, t.Name),
		"PK column %q does not match expected %s", pkCol, strings.Join(expected, " or "))
}

// ExpectedPKName returns expected PK column names for a table based on naming convention.
func ExpectedPKName(tableName, naming string) []string {
	singular := singularize(tableName)
	if singular == "" {
		return nil
	}

	var names []string
	switch naming {
	case "snake_case":
		names = append(names, singular+"_id")
	default: // camelCase, PascalCase
		names = append(names, singular+"Id")
	}

	// for compound names like "showInfos" → also try last-word singular: "showInfoId"
	// and first-word singular: "showId"
	lastWord := extractLastWord(tableName, naming)
	if lastWord != "" {
		lastSingular := singularize(lastWord)
		if lastSingular != "" && lastSingular != singular {
			switch naming {
			case "snake_case":
				names = append(names, lastSingular+"_id")
			default:
				names = append(names, lastSingular+"Id")
			}
		}
	}

	return names
}

// singularize returns singular form in the same case style as input.
func singularize(name string) string {
	if name == "" {
		return ""
	}
	// inflection works with Title case
	result := inflection.Singular(name)
	if result == name {
		// try with first letter uppercased
		titled := string(unicode.ToUpper(rune(name[0]))) + name[1:]
		s := inflection.Singular(titled)
		if s != titled {
			result = string(unicode.ToLower(rune(s[0]))) + s[1:]
		}
	}
	return result
}

// extractLastWord extracts last word from camelCase or snake_case identifier.
func extractLastWord(name, naming string) string {
	if naming == "snake_case" {
		if idx := strings.LastIndex(name, "_"); idx >= 0 {
			return name[idx+1:]
		}
		return ""
	}
	// camelCase: find last uppercase boundary
	lastUpper := -1
	for i := 1; i < len(name); i++ {
		if unicode.IsUpper(rune(name[i])) {
			lastUpper = i
		}
	}
	if lastUpper > 0 {
		return strings.ToLower(name[lastUpper:lastUpper+1]) + name[lastUpper+1:]
	}
	return ""
}

// checkTablePlural checks that table name follows plural/singular convention.
func checkTablePlural(v *validator, schema string, t *pgd.Table) {
	mode := v.p.ProjectMeta.Settings.Naming.Tables
	if mode == "" {
		return
	}

	path := tpath(schema, t.Name)
	lastWord := extractLastWord(t.Name, v.naming)
	word := lastWord
	if word == "" {
		word = t.Name
	}

	switch mode {
	case "plural":
		plural := inflection.Plural(strings.ToLower(word))
		if !strings.EqualFold(word, plural) {
			expected := replaceLastWord(t.Name, word, plural, v.naming)
			v.infof(RuleTablePlural, path, "table %q should be plural %q", t.Name, expected)
		}
	case "singular":
		singular := inflection.Singular(strings.ToLower(word))
		if !strings.EqualFold(word, singular) {
			expected := replaceLastWord(t.Name, word, singular, v.naming)
			v.infof(RuleTablePlural, path, "table %q should be singular %q", t.Name, expected)
		}
	}
}

// replaceLastWord replaces the last word in a compound name, preserving case.
// "objectHistory" + "history" → "histories" = "objectHistories"
func replaceLastWord(name, oldWord, newWord, naming string) string {
	prefix := name[:len(name)-len(oldWord)]
	if naming == "snake_case" {
		return prefix + newWord
	}
	// camelCase: capitalize first letter of replacement
	if len(prefix) > 0 {
		return prefix + string(unicode.ToUpper(rune(newWord[0]))) + newWord[1:]
	}
	return newWord
}

// --- index checks ---

func checkIndexTable(v *validator, schema string, idx *pgd.Index) {
	v.checkIdent(tpath(schema, idx.Name), idx.Name)
	if _, ok := v.tables[tpath(schema, idx.Table)]; !ok {
		v.errorf(RuleIdxTableNotFound, tpath(schema, idx.Name), "index references unknown table %q", idx.Table)
	}
}

func checkIndexColumns(v *validator, schema string, idx *pgd.Index) {
	cols := v.columns[tpath(schema, idx.Table)]
	if cols == nil {
		return // table doesn't exist, E012 already reported
	}
	for _, ref := range idx.Columns {
		if _, ok := cols[ref.Name]; !ok {
			v.errorf(RuleIdxColNotFound, tpath(schema, idx.Name),
				"index references unknown column %q in table %q", ref.Name, idx.Table)
		}
	}
	if idx.Include != nil {
		for _, ref := range idx.Include.Columns {
			if _, ok := cols[ref.Name]; !ok {
				v.errorf(RuleIncludeColNotFound, tpath(schema, idx.Name),
					"INCLUDE references unknown column %q in table %q", ref.Name, idx.Table)
			}
		}
	}
}

func checkIndexOnBoolean(v *validator, schema string, idx *pgd.Index) {
	cols := v.columns[tpath(schema, idx.Table)]
	if cols == nil {
		return
	}
	for _, ref := range idx.Columns {
		if c, ok := cols[ref.Name]; ok {
			lt := strings.ToLower(c.Type)
			if lt == "boolean" || lt == "bool" {
				v.infof(RuleIndexOnBool, tpath(schema, idx.Name),
					"index on boolean column %q — partial index is more efficient", ref.Name)
			}
		}
	}
}

// --- additional project checks ---

func checkOverlappingIndexes(v *validator) {
	for _, s := range v.p.Schemas {
		// group indexes by table
		byTable := make(map[string][]pgd.Index)
		for _, idx := range s.Indexes {
			byTable[idx.Table] = append(byTable[idx.Table], idx)
		}
		for _, idxs := range byTable {
			for i, a := range idxs {
				for j, b := range idxs {
					if i == j || len(a.Columns) == 0 || len(b.Columns) == 0 {
						continue
					}
					// a is redundant if a.Columns is a strict prefix of b.Columns
					if len(a.Columns) < len(b.Columns) && colRefsPrefix(a.Columns, b.Columns) {
						v.warnf(RuleOverlapIndex, tpath(s.Name, a.Name),
							"index is a prefix of %q on table %q — redundant", b.Name, a.Table)
					}
				}
			}
		}
	}
}

func colRefsPrefix(short, long []pgd.ColRef) bool {
	for i, c := range short {
		if c.Name != long[i].Name {
			return false
		}
	}
	return true
}

func checkTooManyIndexes(v *validator) {
	for _, s := range v.p.Schemas {
		count := make(map[string]int)
		for _, idx := range s.Indexes {
			count[idx.Table]++
		}
		for table, n := range count {
			if n > 10 {
				v.infof(RuleTooManyIndexes, tpath(s.Name, table),
					"table has %d indexes — may slow down writes", n)
			}
		}
	}
}

// --- additional table checks ---

func (v *validator) checkFKTargetUnique(schema string, t *pgd.Table, fk *pgd.ForeignKey, target *pgd.Table) {
	refCols := make([]string, len(fk.Columns))
	for i, fc := range fk.Columns {
		refCols[i] = fc.References
	}

	// check PK
	if target.PK != nil && colNamesMatch(target.PK.Columns, refCols) {
		return
	}
	// check UNIQUE constraints
	for _, u := range target.Uniques {
		if colNamesMatch(u.Columns, refCols) {
			return
		}
	}

	v.warnf(RuleFKToNonUnique, tpath(schema, t.Name),
		"FK %q references columns without PK or UNIQUE in %q", fk.Name, fk.ToTable)
}

func colNamesMatch(refs []pgd.ColRef, names []string) bool {
	return len(refs) == len(names) && prefixMatch(refs, names)
}

func checkDuplicateFKs(v *validator, schema string, t *pgd.Table) {
	type fkSig struct{ cols, target string }
	seen := make(map[fkSig]string) // sig → first FK name
	for _, fk := range t.FKs {
		names := make([]string, len(fk.Columns))
		for i, fc := range fk.Columns {
			names[i] = fc.Name
		}
		k := fkSig{strings.Join(names, ","), fk.ToTable}
		if first, ok := seen[k]; ok {
			v.warnf(RuleDupFK, tpath(schema, t.Name),
				"FK %q is a duplicate of %q (same columns, same target)", fk.Name, first)
		} else {
			seen[k] = fk.Name
		}
	}
}

func checkMultiIdentity(v *validator, schema string, t *pgd.Table) {
	var count int
	for _, c := range t.Columns {
		if c.Identity != nil {
			count++
		}
	}
	if count > 1 {
		v.errorf(RuleMultiIdentity, tpath(schema, t.Name),
			"table has %d identity columns (PostgreSQL allows only one)", count)
	}
}

func checkReservedWords(v *validator, schema string, t *pgd.Table) {
	path := tpath(schema, t.Name)
	if pgReservedWords[strings.ToLower(t.Name)] {
		v.warnf(RuleReservedWord, path, "%q is a PostgreSQL reserved word", t.Name)
	}
	for _, c := range t.Columns {
		if pgReservedWords[strings.ToLower(c.Name)] {
			v.warnf(RuleReservedWord, cpath(schema, t.Name, c.Name), "%q is a PostgreSQL reserved word", c.Name)
		}
	}
}

// --- version-aware checks ---

func (v *validator) versionWarn(path string, minPG int, feature string) {
	if v.pgVersion > 0 && v.pgVersion < minPG {
		v.warnf(RuleVersionFeature, path, "%s requires PostgreSQL %d+", feature, minPG)
	}
}

func checkVersionFeatures(v *validator, schema string, t *pgd.Table) {
	path := tpath(schema, t.Name)
	for _, c := range t.Columns {
		cp := cpath(schema, t.Name, c.Name)
		if c.Identity != nil {
			v.versionWarn(cp, 10, "identity columns")
		}
		if c.Generated != nil {
			if c.Generated.Stored == "false" {
				v.versionWarn(cp, 18, "GENERATED ALWAYS AS ... VIRTUAL")
			} else {
				v.versionWarn(cp, 12, "GENERATED ALWAYS AS ... STORED")
			}
		}
		if c.Compression != "" {
			v.versionWarn(cp, 14, "column compression (lz4/pglz)")
		}
	}
	if t.PartitionBy != nil {
		v.versionWarn(path, 10, "table partitioning")
	}
	for _, u := range t.Uniques {
		if u.NullsDistinct == "true" {
			v.versionWarn(path, 15, "NULLS NOT DISTINCT")
		}
	}
	for _, ex := range t.Excludes {
		_ = ex
		v.versionWarn(path, 10, "EXCLUDE constraint")
	}
}

func checkIndexVersionFeatures(v *validator, schema string, idx *pgd.Index) {
	path := tpath(schema, idx.Name)
	if idx.Include != nil && len(idx.Include.Columns) > 0 {
		v.versionWarn(path, 11, "INCLUDE columns in index")
	}
}

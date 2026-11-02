// Package gendata generates realistic test data INSERT statements for a pgd Project.
package gendata

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

const (
	DefaultRows      = 50
	DefaultBatchSize = 100
)

// Options configures test data generation.
type Options struct {
	Seed      int64            // 0 = random
	Rows      int              // default rows per table
	BatchSize int              // rows per INSERT statement
	Tables    map[string]Table // per-table overrides, key = table name
}

// Table holds per-table generation settings.
type Table struct {
	Rows    int               // 0 = use default
	Skip    bool              // skip this table entirely
	Columns map[string]ColGen // per-column generator overrides
}

// ColGen overrides the generator for a specific column.
type ColGen struct {
	Generator string // gofakeit function name or special: "pick", "autoincrement"
	Params    string // comma-separated params, e.g. "values=a,b,c" or "min=1,max=100"
}

// tableKey uniquely identifies a table within a project.
type tableKey struct {
	Schema string
	Table  string
}

func (tk tableKey) String() string {
	if tk.Schema == "" || tk.Schema == "public" {
		return tk.Table
	}
	return tk.Schema + "." + tk.Table
}

func (tk tableKey) Qualified() string {
	if tk.Schema == "" || tk.Schema == "public" {
		return pgd.QuoteIdent(tk.Table)
	}
	return pgd.QuoteIdent(tk.Schema) + "." + pgd.QuoteIdent(tk.Table)
}

// uniqueTracker ensures UNIQUE constraint satisfaction.
type uniqueTracker struct {
	// single-column unique: column name → set of used values
	singleUsed map[string]map[string]bool
	// composite unique: constraint name → set of composite key strings
	compositeUsed map[string]map[string]bool
	// column → list of constraint names it participates in (composite only)
	compositeKeys map[string][]compositeUniqueKey
}

type compositeUniqueKey struct {
	name    string   // constraint name
	columns []string // all column names in this constraint
}

func newUniqueTracker(t *pgd.Table) *uniqueTracker {
	ut := &uniqueTracker{
		singleUsed:    make(map[string]map[string]bool),
		compositeUsed: make(map[string]map[string]bool),
		compositeKeys: make(map[string][]compositeUniqueKey),
	}
	// add PK as unique constraint
	if t.PK != nil {
		if len(t.PK.Columns) == 1 {
			ut.singleUsed[t.PK.Columns[0].Name] = make(map[string]bool)
		} else {
			key := compositeUniqueKey{name: "_pk_", columns: make([]string, len(t.PK.Columns))}
			for i, c := range t.PK.Columns {
				key.columns[i] = c.Name
			}
			ut.compositeUsed["_pk_"] = make(map[string]bool)
			for _, c := range t.PK.Columns {
				ut.compositeKeys[c.Name] = append(ut.compositeKeys[c.Name], key)
			}
		}
	}

	for _, u := range t.Uniques {
		if len(u.Columns) == 1 {
			ut.singleUsed[u.Columns[0].Name] = make(map[string]bool)
		} else {
			key := compositeUniqueKey{name: u.Name, columns: make([]string, len(u.Columns))}
			for i, c := range u.Columns {
				key.columns[i] = c.Name
			}
			ut.compositeUsed[u.Name] = make(map[string]bool)
			for _, c := range u.Columns {
				ut.compositeKeys[c.Name] = append(ut.compositeKeys[c.Name], key)
			}
		}
	}
	return ut
}

// isSingleUnique returns true if column has a single-column UNIQUE constraint.
func (ut *uniqueTracker) isSingleUnique(col string) bool {
	_, ok := ut.singleUsed[col]
	return ok
}

// checkAndTrackSingle checks if val is unique for col, tracks it, returns true if ok.
func (ut *uniqueTracker) checkAndTrackSingle(col, val string) bool {
	used, ok := ut.singleUsed[col]
	if !ok {
		return true // no constraint
	}
	if used[val] {
		return false
	}
	used[val] = true
	return true
}

// checkAndTrackComposite checks all composite UNIQUE constraints using the row's values.
// Returns true if all composite constraints are satisfied.
func (ut *uniqueTracker) checkAndTrackComposite(rowVals map[string]string) bool {
	// collect all composite keys to check
	checked := make(map[string]bool)
	for name, used := range ut.compositeUsed {
		// find the key for this constraint
		var key compositeUniqueKey
		for _, keys := range ut.compositeKeys {
			for _, k := range keys {
				if k.name == name {
					key = k
					break
				}
			}
			if key.name != "" {
				break
			}
		}
		if key.name == "" {
			continue
		}
		// build composite value
		parts := make([]string, len(key.columns))
		allPresent := true
		for i, col := range key.columns {
			v, ok := rowVals[col]
			if !ok {
				allPresent = false
				break
			}
			parts[i] = v
		}
		if !allPresent {
			continue
		}
		composite := strings.Join(parts, "|")
		if used[composite] {
			return false
		}
		checked[name] = true
	}
	// all ok — track them
	for name := range checked {
		var key compositeUniqueKey
		for _, keys := range ut.compositeKeys {
			for _, k := range keys {
				if k.name == name {
					key = k
					break
				}
			}
			if key.name != "" {
				break
			}
		}
		parts := make([]string, len(key.columns))
		for i, col := range key.columns {
			parts[i] = rowVals[col]
		}
		ut.compositeUsed[name][strings.Join(parts, "|")] = true
	}
	return true
}

// deferredUpdate stores an UPDATE to fix circular FK references.
type deferredUpdate struct {
	table tableKey
	pkCol string
	pkVal string
	fkCol string
	fkVal string
}

// Generate writes test data INSERT statements to w for the given project.
//
//nolint:gocognit,gocyclo,cyclop // orchestrates full generation pipeline
func Generate(w io.Writer, p *pgd.Project, opts Options) error {
	if opts.Rows <= 0 {
		opts.Rows = DefaultRows
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = DefaultBatchSize
	}

	faker := gofakeit.New(uint64(opts.Seed))

	// collect all tables
	tables, tableMap := collectTables(p)

	// build FK graph and topological sort
	graph := buildFKGraph(tables, tableMap)
	order, cycleEdges := topoSortFixed(graph, tables)

	// build enum map for enum-aware generation
	enums := collectEnums(p)

	// build domain map: domain name → base type
	domains := collectDomains(p)

	// build domain range constraints (e.g. year >= 1901 AND year <= 2155)
	domainRanges := collectDomainRanges(p)

	// track generated PK values per table
	generatedPKs := make(map[tableKey][]string)

	fmt.Fprintf(w, "-- Generated by pgdesigner test data generator\n")
	fmt.Fprintf(w, "-- Seed: %d\n\n", opts.Seed)
	fmt.Fprintf(w, "BEGIN;\n\n")

	var deferred []deferredUpdate

	for _, tk := range order {
		ti := tableMap[tk]
		t := ti.table
		schema := ti.schema

		rowCount := opts.Rows
		if to, ok := opts.Tables[tk.Table]; ok {
			if to.Skip {
				continue
			}
			if to.Rows > 0 {
				rowCount = to.Rows
			}
		}
		if to, ok := opts.Tables[tk.String()]; ok {
			if to.Skip {
				continue
			}
			if to.Rows > 0 {
				rowCount = to.Rows
			}
		}

		// determine which columns to generate
		cols := selectColumns(t)
		if len(cols) == 0 {
			continue
		}

		// find PK column index for tracking
		pkColName := findPKColumn(t)

		// find FK mappings: local column name → parent table key
		fkMap := buildFKMap(t, schema)

		// find cycle-broken FK columns
		brokenFKs := make(map[string]bool)
		for _, ce := range cycleEdges {
			if ce.from == tk {
				for _, fk := range t.FKs {
					target := resolveFK(fk, schema)
					if target == ce.to {
						for _, fc := range fk.Columns {
							brokenFKs[fc.Name] = true
						}
					}
				}
			}
		}

		// column overrides
		var colOverrides map[string]ColGen
		if to, ok := opts.Tables[tk.Table]; ok {
			colOverrides = to.Columns
		}
		if to, ok := opts.Tables[tk.String()]; ok {
			colOverrides = to.Columns
		}

		// build CHECK maps for this table
		checkIN := parseCheckINValues(t.Checks)
		checkRange := parseCheckRanges(t.Checks)

		// build partition constraints
		partRange, partListIN := buildPartitionConstraints(t)

		// unique constraint tracker
		uTracker := newUniqueTracker(t)

		fmt.Fprintln(w, pgd.SQLComment("Table: %s (%d rows)", tk.Qualified(), rowCount))

		hasIdentity := hasIdentityColumn(t)

		// generate all rows first, then write in batches
		type rowData struct {
			vals  []string
			pkVal string
		}
		var allRows []rowData
		var autoInc int

		for i := range rowCount {
			autoInc++

			ctx := &genContext{
				tableName:    strings.ToLower(t.Name),
				enums:        enums,
				domains:      domains,
				domainRanges: domainRanges,
				checkIN:      checkIN,
				checkRange:   checkRange,
				overrides:    colOverrides,
				rowVals:      make(map[string]string),
				partRange:    partRange,
				partListIN:   partListIN,
			}

			vals := make([]string, len(cols))
			var pkVal string

			for j, c := range cols {
				if brokenFKs[c.Name] {
					vals[j] = "NULL"
					continue
				}

				if parentTK, ok := fkMap[c.Name]; ok { //nolint:nestif // FK resolution logic
					if c.Name == pkColName {
						// PK is also FK — generate normally
					} else if pks, ok := generatedPKs[parentTK]; ok && len(pks) > 0 {
						if parentTK == tk && i < rowCount*3/10 && c.Nullable != "false" {
							vals[j] = "NULL"
							continue
						}
						vals[j] = pks[faker.IntRange(0, len(pks)-1)]
						continue
					} else if c.Nullable != "false" {
						vals[j] = "NULL"
						continue
					}
				}

				vals[j] = generateValue(faker, c, autoInc, ctx)

				// ensure single-column UNIQUE via suffix
				if uTracker.isSingleUnique(c.Name) && !uTracker.checkAndTrackSingle(c.Name, vals[j]) {
					suffix := fmt.Sprintf("_%d", autoInc)
					if isQuotedString(vals[j]) {
						inner := vals[j][1 : len(vals[j])-1]
						ml := maxLen(c)
						if ml > 0 {
							inner = truncate(inner, ml-len(suffix))
						}
						vals[j] = sqlQuote(inner + suffix)
					} else {
						vals[j] += suffix
					}
					uTracker.checkAndTrackSingle(c.Name, vals[j])
				}

				if c.Name == pkColName {
					pkVal = vals[j]
				}
			}

			// composite UNIQUE check
			compositeVals := make(map[string]string, len(cols))
			for j, c := range cols {
				compositeVals[c.Name] = vals[j]
			}
			if !uTracker.checkAndTrackComposite(compositeVals) {
				continue // skip duplicate row
			}

			allRows = append(allRows, rowData{vals: vals, pkVal: pkVal})

			if pkVal != "" {
				generatedPKs[tk] = append(generatedPKs[tk], pkVal)
			}

			// deferred FK updates for broken cycles
			if pkVal != "" && len(brokenFKs) > 0 { //nolint:nestif // deferred update collection
				for fkCol := range brokenFKs {
					if parentTK, ok := fkMap[fkCol]; ok {
						for _, c := range cols {
							if c.Name == fkCol && c.Nullable != "false" {
								if pks, ok := generatedPKs[parentTK]; ok && len(pks) > 0 {
									deferred = append(deferred, deferredUpdate{
										table: tk,
										pkCol: pkColName,
										pkVal: pkVal,
										fkCol: fkCol,
										fkVal: pks[faker.IntRange(0, len(pks)-1)],
									})
								}
							}
						}
					}
				}
			}
		}

		// write rows in batches
		colNames := make([]string, len(cols))
		for i, c := range cols {
			colNames[i] = pgd.QuoteIdent(c.Name)
		}
		overriding := ""
		if hasIdentity {
			overriding = " OVERRIDING SYSTEM VALUE"
		}

		for batchStart := 0; batchStart < len(allRows); batchStart += opts.BatchSize {
			batchEnd := batchStart + opts.BatchSize
			if batchEnd > len(allRows) {
				batchEnd = len(allRows)
			}
			fmt.Fprintf(w, "INSERT INTO %s (%s)%s VALUES\n", tk.Qualified(), strings.Join(colNames, ", "), overriding)
			for j := batchStart; j < batchEnd; j++ {
				sep := ","
				if j == batchEnd-1 {
					sep = ";"
				}
				fmt.Fprintf(w, "  (%s)%s\n", strings.Join(allRows[j].vals, ", "), sep)
			}
			fmt.Fprintln(w)
		}
	}

	// write deferred FK updates
	if len(deferred) > 0 {
		fmt.Fprintf(w, "-- Deferred FK updates (circular references)\n")
		for _, d := range deferred {
			fmt.Fprintf(w, "UPDATE %s SET %s = %s WHERE %s = %s;\n",
				d.table.Qualified(),
				pgd.QuoteIdent(d.fkCol), d.fkVal,
				pgd.QuoteIdent(d.pkCol), d.pkVal)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "COMMIT;\n")
	return nil
}

// tableInfo stores a table reference with its schema context.
type tableInfo struct {
	table  *pgd.Table
	schema string
	key    tableKey
}

// collectTables returns all tables and a lookup map.
func collectTables(p *pgd.Project) ([]tableKey, map[tableKey]*tableInfo) {
	var keys []tableKey
	m := make(map[tableKey]*tableInfo)
	for i := range p.Schemas {
		s := &p.Schemas[i]
		for j := range s.Tables {
			t := &s.Tables[j]
			if t.Generate == "false" || t.PartitionOf != "" {
				continue
			}
			tk := tableKey{Schema: s.Name, Table: t.Name}
			keys = append(keys, tk)
			m[tk] = &tableInfo{table: t, schema: s.Name, key: tk}
		}
	}
	return keys, m
}

// edge represents a directed FK dependency.
type edge struct {
	from, to tableKey
}

// buildFKGraph builds a dependency graph: for each table, which tables it depends on.
func buildFKGraph(tables []tableKey, tableMap map[tableKey]*tableInfo) map[tableKey][]tableKey {
	graph := make(map[tableKey][]tableKey)
	for _, tk := range tables {
		graph[tk] = nil // ensure every table is in graph
		ti := tableMap[tk]
		for _, fk := range ti.table.FKs {
			target := resolveFK(fk, ti.schema)
			if _, ok := tableMap[target]; ok {
				graph[tk] = append(graph[tk], target)
			}
		}
	}
	return graph
}

// resolveFK determines the target tableKey for a foreign key.
func resolveFK(fk pgd.ForeignKey, currentSchema string) tableKey {
	parts := strings.SplitN(fk.ToTable, ".", 2)
	if len(parts) == 2 {
		return tableKey{Schema: parts[0], Table: parts[1]}
	}
	return tableKey{Schema: currentSchema, Table: fk.ToTable}
}

// topoSortFixed sorts tables so parents come before children (Kahn's algorithm).
func topoSortFixed(graph map[tableKey][]tableKey, allTables []tableKey) ([]tableKey, []edge) {
	// graph[child] = [parent1, parent2, ...] means child depends on parents
	// We need parents first. In-degree = number of dependencies a table has.
	inDeg := make(map[tableKey]int)
	for _, tk := range allTables {
		inDeg[tk] = 0
	}

	// reverse adjacency: parent → [children that depend on it]
	reverseAdj := make(map[tableKey][]tableKey)
	for child, parents := range graph {
		for _, parent := range parents {
			if child == parent {
				continue // skip self-ref
			}
			reverseAdj[parent] = append(reverseAdj[parent], child)
			inDeg[child]++
		}
	}

	// Kahn's: start with tables that have no dependencies
	// Use sorted queue for deterministic output
	var queue []tableKey
	for _, tk := range allTables {
		if inDeg[tk] == 0 {
			queue = append(queue, tk)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i].String() < queue[j].String() })

	var result []tableKey
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		var ready []tableKey
		for _, child := range reverseAdj[node] {
			inDeg[child]--
			if inDeg[child] == 0 {
				ready = append(ready, child)
			}
		}
		sort.Slice(ready, func(i, j int) bool { return ready[i].String() < ready[j].String() })
		queue = append(queue, ready...)
	}

	// handle cycles
	var cycleEdges []edge
	if len(result) < len(allTables) {
		inResult := make(map[tableKey]bool, len(result))
		for _, tk := range result {
			inResult[tk] = true
		}
		for _, tk := range allTables {
			if !inResult[tk] {
				// break one dependency edge
				for _, parent := range graph[tk] {
					if parent != tk && !inResult[parent] {
						cycleEdges = append(cycleEdges, edge{from: tk, to: parent})
						break
					}
				}
				result = append(result, tk)
				inResult[tk] = true
			}
		}
	}

	return result, cycleEdges
}

// selectColumns returns columns that should be included in INSERT.
// Identity columns are included so we can track PK values for FK references.
func selectColumns(t *pgd.Table) []pgd.Column {
	var cols []pgd.Column
	for _, c := range t.Columns {
		// skip generated columns (STORED/VIRTUAL)
		if c.Generated != nil {
			continue
		}
		// skip serial types (DB auto-generates)
		if isSerial(c.Type) {
			continue
		}
		// include identity columns (we generate explicit values)
		cols = append(cols, c)
	}
	return cols
}

// hasIdentityColumn returns true if the table has any identity column.
func hasIdentityColumn(t *pgd.Table) bool {
	for _, c := range t.Columns {
		if c.Identity != nil {
			return true
		}
	}
	return false
}

// findPKColumn returns the first PK column name, or "" if composite/missing.
func findPKColumn(t *pgd.Table) string {
	if t.PK != nil && len(t.PK.Columns) == 1 {
		return t.PK.Columns[0].Name
	}
	// check for identity column as implicit PK
	for _, c := range t.Columns {
		if c.Identity != nil {
			return c.Name
		}
	}
	return ""
}

// buildFKMap returns a map of local column name → target table key.
func buildFKMap(t *pgd.Table, schema string) map[string]tableKey {
	m := make(map[string]tableKey)
	for _, fk := range t.FKs {
		target := resolveFK(fk, schema)
		for _, fc := range fk.Columns {
			m[fc.Name] = target
		}
	}
	return m
}

// collectEnums builds a map of enum type name → labels.
func collectEnums(p *pgd.Project) map[string][]string {
	m := make(map[string][]string)
	if p.Types == nil {
		return m
	}
	for _, e := range p.Types.Enums {
		m[e.Name] = e.Labels
		if e.Schema != "" && e.Schema != "public" {
			m[e.Schema+"."+e.Name] = e.Labels
		}
	}
	return m
}

// collectDomainRanges parses domain CHECK constraints for integer range bounds.
// Matches patterns like: value >= N AND value <= M
func collectDomainRanges(p *pgd.Project) map[string]intRange {
	m := make(map[string]intRange)
	if p.Types == nil {
		return m
	}
	re := regexp.MustCompile(`(?i)value\s*>=\s*(\d+)\s+AND\s+value\s*<=\s*(\d+)`)
	for _, d := range p.Types.Domains {
		for _, c := range d.Constraints {
			matches := re.FindStringSubmatch(c.Expression)
			if len(matches) == 3 {
				var lo, hi int
				if _, err := fmt.Sscan(matches[1], &lo); err == nil {
					if _, err := fmt.Sscan(matches[2], &hi); err == nil {
						m[d.Name] = intRange{min: lo, max: hi}
					}
				}
			}
		}
	}
	return m
}

// collectDomains builds a map of domain name → base PG type.
func collectDomains(p *pgd.Project) map[string]string {
	m := make(map[string]string)
	if p.Types == nil {
		return m
	}
	for _, d := range p.Types.Domains {
		m[d.Name] = d.Type
		if d.Schema != "" && d.Schema != "public" {
			m[d.Schema+"."+d.Name] = d.Type
		}
	}
	return m
}

// buildPartitionConstraints extracts partition key constraints from table partitions.
// For RANGE partitions: extracts min/max time bounds.
// For LIST partitions: extracts allowed values.
func buildPartitionConstraints(t *pgd.Table) (map[string]timeRange, map[string][]string) {
	partRange := make(map[string]timeRange)
	partListIN := make(map[string][]string)

	if t.PartitionBy == nil || len(t.Partitions) == 0 || len(t.PartitionBy.Columns) == 0 {
		return partRange, partListIN
	}

	colName := t.PartitionBy.Columns[0].Name

	switch strings.ToLower(t.PartitionBy.Type) {
	case "range":
		var globalMin, globalMax time.Time
		first := true
		for _, p := range t.Partitions {
			lo, hi := parseRangeBound(p.Bound)
			if lo.IsZero() || hi.IsZero() {
				continue
			}
			if first || lo.Before(globalMin) {
				globalMin = lo
			}
			if first || hi.After(globalMax) {
				globalMax = hi
			}
			first = false
		}
		if !globalMin.IsZero() && !globalMax.IsZero() {
			partRange[colName] = timeRange{min: globalMin, max: globalMax.Add(-time.Second)}
		}

	case "list":
		var vals []string
		for _, p := range t.Partitions {
			vals = append(vals, parseListBound(p.Bound)...)
		}
		if len(vals) > 0 {
			partListIN[colName] = vals
		}
	}

	return partRange, partListIN
}

// rangeBoundRe matches: FOR VALUES FROM ('...') TO ('...')
var rangeBoundRe = regexp.MustCompile(`(?i)FOR\s+VALUES\s+FROM\s*\('([^']+)'\)\s*TO\s*\('([^']+)'\)`)

// parseRangeBound extracts FROM and TO timestamps from a RANGE partition bound.
func parseRangeBound(bound string) (time.Time, time.Time) {
	m := rangeBoundRe.FindStringSubmatch(bound)
	if len(m) < 3 {
		return time.Time{}, time.Time{}
	}
	lo := parseTimestamp(m[1])
	hi := parseTimestamp(m[2])
	return lo, hi
}

// parseTimestamp tries several timestamp formats.
func parseTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05+07",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05+07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// listBoundRe matches: FOR VALUES IN ('val1', 'val2', ...)
var listBoundRe = regexp.MustCompile(`(?i)FOR\s+VALUES\s+IN\s*\(([^)]+)\)`)

// parseListBound extracts values from a LIST partition bound.
func parseListBound(bound string) []string {
	m := listBoundRe.FindStringSubmatch(bound)
	if len(m) < 2 {
		return nil
	}
	var vals []string
	for _, part := range strings.Split(m[1], ",") {
		part = strings.TrimSpace(part)
		if len(part) >= 2 && part[0] == '\'' && part[len(part)-1] == '\'' {
			vals = append(vals, part[1:len(part)-1])
		}
	}
	return vals
}

func isSerial(t string) bool {
	switch strings.ToLower(t) {
	case "serial", "bigserial", "smallserial":
		return true
	}
	return false
}

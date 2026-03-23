// Package merge combines two pgd.Project files into one with maximum coverage.
package merge

import (
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Options configures merge behavior.
type Options struct {
	Layout string // "both" (default), "base", "overlay"
	Name   string // override project name (empty = use overlay name)
}

// Result holds merge statistics.
type Result struct {
	Common      int
	OnlyBase    int
	OnlyOverlay int
	Total       int
}

// Merge combines base and overlay projects. Overlay wins on conflicts.
func Merge(base, overlay *pgd.Project, opts Options) (*pgd.Project, Result) {
	if opts.Layout == "" {
		opts.Layout = "both"
	}

	p := &pgd.Project{
		Version:       max(base.Version, overlay.Version),
		PgVersion:     maxPgVersion(base.PgVersion, overlay.PgVersion),
		DefaultSchema: overlay.DefaultSchema,
		ProjectMeta:   overlay.ProjectMeta,
	}
	if opts.Name != "" {
		p.ProjectMeta.Name = opts.Name
	}
	if p.DefaultSchema == "" {
		p.DefaultSchema = base.DefaultSchema
	}

	// extensions, sequences, functions, triggers — union by name
	p.Extensions = unionByName(base.Extensions, overlay.Extensions, extName, extEqual)
	p.Sequences = unionByName(base.Sequences, overlay.Sequences, seqName, seqEqual)
	p.Functions = unionByName(base.Functions, overlay.Functions, funcName, funcEqual)
	p.Triggers = unionByName(base.Triggers, overlay.Triggers, trigName, trigEqual)

	// types
	p.Types = mergeTypes(base.Types, overlay.Types)

	// views
	p.Views = mergeViews(base.Views, overlay.Views)

	// comments — overlay wins, union
	p.Comments = unionByName(base.Comments, overlay.Comments, commentKey, commentEqual)

	// schemas + tables + indexes
	var stats Result
	p.Schemas = mergeSchemas(base.Schemas, overlay.Schemas, &stats)

	// layouts
	p.Layouts = mergeLayouts(base, overlay, opts.Layout)

	stats.Total = stats.Common + stats.OnlyBase + stats.OnlyOverlay
	return p, stats
}

// mergeSchemas merges schema lists. Tables within same schema are merged.
func mergeSchemas(base, overlay []pgd.Schema, stats *Result) []pgd.Schema {
	baseMap := indexByName(base, func(s pgd.Schema) string { return s.Name })
	overlayMap := indexByName(overlay, func(s pgd.Schema) string { return s.Name })

	// collect all schema names, overlay order first
	seen := map[string]bool{}
	var names []string
	for _, s := range overlay {
		if !seen[s.Name] {
			names = append(names, s.Name)
			seen[s.Name] = true
		}
	}
	for _, s := range base {
		if !seen[s.Name] {
			names = append(names, s.Name)
			seen[s.Name] = true
		}
	}

	var result []pgd.Schema
	for _, name := range names {
		bs, bOK := baseMap[name]
		os, oOK := overlayMap[name]

		switch {
		case bOK && oOK:
			result = append(result, mergeSchema(bs, os, stats))
		case oOK:
			stats.OnlyOverlay += len(os.Tables)
			result = append(result, os)
		case bOK:
			stats.OnlyBase += len(bs.Tables)
			result = append(result, bs)
		}
	}
	return result
}

func mergeSchema(base, overlay pgd.Schema, stats *Result) pgd.Schema {
	s := pgd.Schema{Name: overlay.Name}

	baseTableMap := indexByName(base.Tables, func(t pgd.Table) string { return t.Name })

	// overlay tables first (overlay order)
	seen := map[string]bool{}
	for _, ot := range overlay.Tables {
		seen[ot.Name] = true
		if bt, ok := baseTableMap[ot.Name]; ok {
			s.Tables = append(s.Tables, mergeTable(bt, ot))
			stats.Common++
		} else {
			s.Tables = append(s.Tables, ot)
			stats.OnlyOverlay++
		}
	}
	// base-only tables
	for _, bt := range base.Tables {
		if !seen[bt.Name] {
			s.Tables = append(s.Tables, bt)
			stats.OnlyBase++
		}
	}

	// indexes — union by name
	s.Indexes = unionByName(base.Indexes, overlay.Indexes, idxName, idxEqual)

	return s
}

// mergeTable merges two tables with same name. Overlay wins on conflicts.
func mergeTable(base, overlay pgd.Table) pgd.Table {
	t := overlay // start with overlay

	// merge columns: overlay order, then base-only columns appended
	baseColMap := indexByName(base.Columns, func(c pgd.Column) string { return c.Name })
	seen := map[string]bool{}
	for _, c := range overlay.Columns {
		seen[c.Name] = true
	}
	// append base-only columns at the end
	for _, bc := range base.Columns {
		if !seen[bc.Name] {
			t.Columns = append(t.Columns, bc)
		}
	}

	// PK: overlay wins if set
	if t.PK == nil && base.PK != nil {
		t.PK = base.PK
	}

	// uniques, checks, excludes, FKs — union by name
	t.Uniques = unionByName(base.Uniques, overlay.Uniques, uqName, uqEqual)
	t.Checks = unionByName(base.Checks, overlay.Checks, chkName, chkEqual)
	t.Excludes = unionByName(base.Excludes, overlay.Excludes, exclName, exclEqual)
	t.FKs = unionByName(base.FKs, overlay.FKs, fkName, fkEqual)

	// partitioning: take partitioned side; if both — merge children by name
	t.PartitionBy, t.Partitions = mergePartitioning(base, overlay)

	// comment: overlay wins if non-empty
	if t.Comment == "" && base.Comment != "" {
		t.Comment = base.Comment
	}

	_ = baseColMap // used via seen
	return t
}

// mergeLayouts combines layout positions from both projects.
func mergeLayouts(base, overlay *pgd.Project, mode string) pgd.Layouts {
	basePos := layoutPositions(base)
	overlayPos := layoutPositions(overlay)

	// determine which positions to use
	merged := map[string]pgd.LayoutEntity{}
	switch mode {
	case "base":
		for k, v := range overlayPos {
			merged[k] = v
		}
		for k, v := range basePos {
			merged[k] = v // base overwrites overlay
		}
	case "overlay":
		for k, v := range basePos {
			merged[k] = v
		}
		for k, v := range overlayPos {
			merged[k] = v // overlay overwrites base
		}
	default: // "both" — take from wherever available, overlay wins on conflict
		for k, v := range basePos {
			merged[k] = v
		}
		for k, v := range overlayPos {
			merged[k] = v // overlay overwrites
		}
	}

	var entities []pgd.LayoutEntity
	for _, e := range merged {
		entities = append(entities, e)
	}

	return pgd.Layouts{Layouts: []pgd.Layout{{
		Name: "Default Diagram", Default: "true", Entities: entities,
	}}}
}

func layoutPositions(p *pgd.Project) map[string]pgd.LayoutEntity {
	pos := map[string]pgd.LayoutEntity{}
	for _, l := range p.Layouts.Layouts {
		for _, e := range l.Entities {
			key := e.Schema + "." + e.Table
			if e.Schema == "" {
				key = e.Table
			}
			pos[key] = e
		}
	}
	return pos
}

// mergeTypes merges type definitions (enums, domains, composites).
func mergeTypes(base, overlay *pgd.Types) *pgd.Types {
	if base == nil && overlay == nil {
		return nil
	}
	var bEnums, oEnums []pgd.Enum
	var bDomains, oDomains []pgd.Domain
	var bComps, oComps []pgd.Composite
	if base != nil {
		bEnums, bDomains, bComps = base.Enums, base.Domains, base.Composites
	}
	if overlay != nil {
		oEnums, oDomains, oComps = overlay.Enums, overlay.Domains, overlay.Composites
	}

	enums := unionByName(bEnums, oEnums, enumName, enumEqual)
	domains := unionByName(bDomains, oDomains, domainName, domainEqual)
	comps := unionByName(bComps, oComps, compName, compEqual)

	if len(enums) == 0 && len(domains) == 0 && len(comps) == 0 {
		return nil
	}
	return &pgd.Types{Enums: enums, Domains: domains, Composites: comps}
}

// mergeViews merges view definitions.
func mergeViews(base, overlay *pgd.Views) *pgd.Views {
	if base == nil && overlay == nil {
		return nil
	}
	var bViews, oViews []pgd.View
	var bMat, oMat []pgd.MaterializedView
	if base != nil {
		bViews, bMat = base.Views, base.MatViews
	}
	if overlay != nil {
		oViews, oMat = overlay.Views, overlay.MatViews
	}

	views := unionByName(bViews, oViews, viewName, viewEqual)
	matViews := unionByName(bMat, oMat, matViewName, matViewEqual)

	if len(views) == 0 && len(matViews) == 0 {
		return nil
	}
	return &pgd.Views{Views: views, MatViews: matViews}
}

// generic helpers

// unionByName merges two slices by name. Overlay wins when both exist.
func unionByName[T any](base, overlay []T, name func(T) string, equal func(T, T) bool) []T {
	baseMap := indexByName(base, name)
	seen := map[string]bool{}

	var result []T
	// overlay items first
	for _, o := range overlay {
		n := name(o)
		seen[n] = true
		result = append(result, o)
	}
	// base-only items
	for _, b := range base {
		n := name(b)
		if !seen[n] {
			result = append(result, b)
		}
	}

	_ = baseMap // used via seen
	_ = equal   // reserved for future conflict reporting
	return result
}

func indexByName[T any](items []T, name func(T) string) map[string]T {
	m := make(map[string]T, len(items))
	for _, item := range items {
		m[name(item)] = item
	}
	return m
}

func maxPgVersion(a, b string) string {
	if a > b {
		return a
	}
	return b
}

// mergePartitioning merges partition-by and partitions from two tables.
// If both have PartitionBy with same strategy+columns, children are merged by name.
// If only one side is partitioned, that side wins (additive).
func mergePartitioning(base, overlay pgd.Table) (*pgd.PartitionBy, []pgd.Partition) {
	switch {
	case overlay.PartitionBy != nil && base.PartitionBy != nil:
		// Both partitioned — use overlay strategy, merge children
		parts := unionByName(base.Partitions, overlay.Partitions, partName, partEqual)
		return overlay.PartitionBy, parts
	case overlay.PartitionBy != nil:
		return overlay.PartitionBy, overlay.Partitions
	case base.PartitionBy != nil:
		return base.PartitionBy, base.Partitions
	default:
		return nil, nil
	}
}

func partName(p pgd.Partition) string   { return p.Name }
func partEqual(_, _ pgd.Partition) bool { return true }

// name accessors

func extName(e pgd.Extension) string            { return e.Name }
func seqName(s pgd.Sequence) string             { return s.Name }
func funcName(f pgd.Function) string            { return f.Name }
func trigName(t pgd.Trigger) string             { return t.Name }
func idxName(i pgd.Index) string                { return i.Name }
func uqName(u pgd.Unique) string                { return u.Name }
func chkName(c pgd.Check) string                { return c.Name }
func exclName(e pgd.Exclude) string             { return e.Name }
func fkName(f pgd.ForeignKey) string            { return f.Name }
func enumName(e pgd.Enum) string                { return e.Name }
func domainName(d pgd.Domain) string            { return d.Name }
func compName(c pgd.Composite) string           { return c.Name }
func viewName(v pgd.View) string                { return v.Name }
func matViewName(m pgd.MaterializedView) string { return m.Name }
func commentKey(c pgd.Comment) string           { return c.On + ":" + c.Schema + "." + c.Table + "." + c.Name }

// equal stubs (for future conflict reporting)

func extEqual(_, _ pgd.Extension) bool            { return true }
func seqEqual(_, _ pgd.Sequence) bool             { return true }
func funcEqual(_, _ pgd.Function) bool            { return true }
func trigEqual(_, _ pgd.Trigger) bool             { return true }
func idxEqual(_, _ pgd.Index) bool                { return true }
func uqEqual(_, _ pgd.Unique) bool                { return true }
func chkEqual(_, _ pgd.Check) bool                { return true }
func exclEqual(_, _ pgd.Exclude) bool             { return true }
func fkEqual(_, _ pgd.ForeignKey) bool            { return true }
func enumEqual(_, _ pgd.Enum) bool                { return true }
func domainEqual(_, _ pgd.Domain) bool            { return true }
func compEqual(_, _ pgd.Composite) bool           { return true }
func viewEqual(_, _ pgd.View) bool                { return true }
func matViewEqual(_, _ pgd.MaterializedView) bool { return true }
func commentEqual(_, _ pgd.Comment) bool          { return true }

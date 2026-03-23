// Package pdd converts MicroOLAP Database Designer .pdd files to pgd.Project.
package pdd

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Convert parses a MicroOLAP .pdd XML file and returns a pgd.Project.
// The name parameter is ignored (project name is taken from the .pdd file).
func Convert(data []byte, name string) (*pgd.Project, error) {
	data = fixPDDXML(data)

	var pdd PDD
	if err := xml.Unmarshal(data, &pdd); err != nil {
		return nil, fmt.Errorf("parsing pdd XML: %w", err)
	}
	return convert(&pdd), nil
}

// PDD input structs

type PDD struct {
	XMLName       xml.Name    `xml:"DBMODEL"`
	ModelSettings PDDSettings `xml:"MODELSETTINGS"`
	Schemas       []PDDSchema `xml:"SCHEMAS>SCHEMA"`
	Domains       []PDDDomain `xml:"DOMAINS>DOMAIN"`
	Metadata      PDDMetadata `xml:"METADATA"`
}

type PDDSettings struct {
	ModelName string `xml:"ModelName,attr"`
}

type PDDSchema struct {
	ID   int    `xml:"ID,attr"`
	Name string `xml:"Name,attr"`
}

type PDDDomain struct {
	ID      int    `xml:"ID,attr"`
	Name    string `xml:"Name,attr"`
	Type    string `xml:"Type,attr"`
	Width   int    `xml:"Width,attr"`
	Prec    int    `xml:"Prec,attr"`
	NotNull int    `xml:"NotNull,attr"`
}

type PDDMetadata struct {
	Entities   []PDDEntity    `xml:"ENTITIES>ENTITY"`
	References []PDDReference `xml:"REFERENCES>REFERENCE"`
}

type PDDEntity struct {
	ID          int             `xml:"ID,attr"`
	Name        string          `xml:"Name,attr"`
	SchemaName  string          `xml:"SchemaName,attr"`
	XPos        int             `xml:"XPos,attr"`
	YPos        int             `xml:"YPos,attr"`
	Unlogged    int             `xml:"Unlogged,attr"`
	FillColor   int             `xml:"FillColor,attr"`
	Comments    string          `xml:"Comments,attr"`
	Generate    int             `xml:"Generate,attr"`
	Columns     []PDDColumn     `xml:"COLUMNS>COLUMN"`
	Constraints []PDDConstraint `xml:"CONSTRAINTS>CONSTRAINT"`
	Indexes     []PDDIndex      `xml:"INDEXES>INDEX"`
}

type PDDColumn struct {
	ID           int    `xml:"ID,attr"`
	Name         string `xml:"Name,attr"`
	Pos          int    `xml:"Pos,attr"`
	Type         string `xml:"Type,attr"`
	Width        int    `xml:"Width,attr"`
	Prec         int    `xml:"Prec,attr"`
	NotNull      int    `xml:"NotNull,attr"`
	AutoInc      int    `xml:"AutoInc,attr"`
	PrimaryKey   int    `xml:"PrimaryKey,attr"`
	DefaultValue string `xml:"DefaultValue,attr"`
	QuoteDefault int    `xml:"QuoteDefault,attr"`
	Collation    string `xml:"Collation,attr"`
	Comments     string `xml:"Comments,attr"`
}

type PDDConstraint struct {
	ID         int      `xml:"ID,attr"`
	Name       string   `xml:"Name,attr"`
	Kind       int      `xml:"Kind,attr"` // 0=CHECK, 1=UNIQUE, 2=PK
	Expression string   `xml:"Expression,attr"`
	RawCols    PDDCText `xml:"CONSTRAINTCOLUMNS"`
}

type PDDCText struct {
	Text string `xml:"COMMATEXT,attr"`
}

type PDDIndex struct {
	ID        int      `xml:"ID,attr"`
	Name      string   `xml:"Name,attr"`
	Unique    int      `xml:"Unique,attr"`
	Method    int      `xml:"Method,attr"`
	Predicate string   `xml:"Predicate,attr"`
	RawCols   PDDCText `xml:"INDEXCOLUMNS"`
	RawSorts  PDDCText `xml:"INDEXSORTS"`
	RawNulls  PDDCText `xml:"INDEXNULLS"`
}

type PDDReference struct {
	ID          int    `xml:"ID,attr"`
	Name        string `xml:"Name,attr"`
	Source      int    `xml:"SOURCE,attr"`
	Destination int    `xml:"DESTINATION,attr"`
	FKIDS       string `xml:"FKIDS,attr"`
	OnDelete    int    `xml:"OnDelete,attr"`
	OnUpdate    int    `xml:"OnUpdate,attr"`
	Deferrable  int    `xml:"Deferrable,attr"`
	Generate    int    `xml:"Generate,attr"`
}

// Converter

func convert(pdd *PDD) *pgd.Project {
	entityByID := make(map[int]*PDDEntity, len(pdd.Metadata.Entities))
	colByID := make(map[int]*PDDColumn)
	for i := range pdd.Metadata.Entities {
		e := &pdd.Metadata.Entities[i]
		entityByID[e.ID] = e
		for j := range e.Columns {
			colByID[e.Columns[j].ID] = &e.Columns[j]
		}
	}

	// Build domain lookup: quoted name → domain name, and convert to pgd domains.
	domainNames := make(map[string]string) // quoted name (e.g. `"Name"`) → domain name
	var pgdDomains []pgd.Domain
	for _, d := range pdd.Domains {
		domainNames[`"`+d.Name+`"`] = d.Name
		baseType := pgd.NormalizeType(d.Type)
		dom := pgd.Domain{Name: d.Name, Type: baseType}
		if d.Width > 0 && pgd.NeedsLength(baseType) {
			dom.Length = d.Width
		}
		if d.NotNull == 1 {
			dom.NotNull = &struct{}{}
		}
		pgdDomains = append(pgdDomains, dom)
	}

	refsByDest := make(map[int][]PDDReference)
	for _, ref := range pdd.Metadata.References {
		refsByDest[ref.Destination] = append(refsByDest[ref.Destination], ref)
	}

	schemaName := "public"
	if len(pdd.Schemas) > 0 {
		schemaName = pdd.Schemas[0].Name
	}

	name := pdd.ModelSettings.ModelName
	if name == "" {
		name = "project"
	}

	schema := pgd.Schema{Name: schemaName}
	var entities []pgd.LayoutEntity

	for i := range pdd.Metadata.Entities {
		e := &pdd.Metadata.Entities[i]
		schema.Tables = append(schema.Tables, convertTable(e, refsByDest[e.ID], entityByID, colByID, domainNames))
		for j := range e.Indexes {
			if idx := convertIndex(e.Name, &e.Indexes[j]); idx != nil {
				schema.Indexes = append(schema.Indexes, *idx)
			}
		}
		entities = append(entities, pgd.LayoutEntity{
			Schema: schemaName, Table: e.Name,
			X: e.XPos, Y: e.YPos, Color: fillColorHex(e.FillColor),
		})
	}

	// Deduplicate index names by prepending table name.
	idxNameCount := make(map[string]int)
	for _, idx := range schema.Indexes {
		idxNameCount[idx.Name]++
	}
	for i := range schema.Indexes {
		if idxNameCount[schema.Indexes[i].Name] > 1 {
			schema.Indexes[i].Name = schema.Indexes[i].Table + "_" + schema.Indexes[i].Name
		}
	}

	var types *pgd.Types
	if len(pgdDomains) > 0 {
		types = &pgd.Types{Domains: pgdDomains}
	}

	return &pgd.Project{
		Version: 1, PgVersion: "18", DefaultSchema: schemaName,
		Types: types,
		ProjectMeta: pgd.ProjectMeta{
			Name: name,
			Settings: pgd.Settings{
				Naming:   pgd.Naming{Convention: "camelCase"},
				Defaults: pgd.Defaults{Nullable: "true", OnDelete: "restrict", OnUpdate: "restrict"},
			},
		},
		Schemas: []pgd.Schema{schema},
		Layouts: pgd.Layouts{Layouts: []pgd.Layout{{
			Name: "Default Diagram", Default: "true", Entities: entities,
		}}},
	}
}

func convertTable(e *PDDEntity, refs []PDDReference, entityByID map[int]*PDDEntity, colByID map[int]*PDDColumn, domainNames map[string]string) pgd.Table {
	t := pgd.Table{Name: e.Name}
	if e.Generate == 0 {
		t.Generate = "false"
	}
	if e.Unlogged == 1 {
		t.Unlogged = "true"
	}
	if e.Comments != "" {
		t.Comment = decodeEscape(e.Comments)
	}

	// columns sorted by position
	cols := sortedColumns(e.Columns)
	for i := range cols {
		t.Columns = append(t.Columns, convertColumn(&cols[i], domainNames))
	}

	// PK
	if names := constraintCols(e, 2, colByID); len(names) > 0 {
		pkName := constraintName(e, 2)
		if pkName == "" {
			pkName = "pk_" + e.Name
		}
		t.PK = &pgd.PrimaryKey{Name: pkName, Columns: pgd.ColRefsFromNames(names)}
	}

	// UNIQUE
	for _, con := range e.Constraints {
		if con.Kind != 1 {
			continue
		}
		names := resolveColNames(&con, colByID)
		name := con.Name
		if name == "" {
			name = "uq_" + e.Name + "_" + strings.Join(names, "_")
		}
		t.Uniques = append(t.Uniques, pgd.Unique{Name: name, Columns: pgd.ColRefsFromNames(names)})
	}

	// CHECK
	for _, con := range e.Constraints {
		if con.Kind != 0 {
			continue
		}
		expr := decodeEscape(con.Expression)
		if expr == "" {
			continue
		}
		name := con.Name
		if name == "" {
			name = "chk_" + e.Name
		}
		t.Checks = append(t.Checks, pgd.Check{Name: name, Expression: expr})
	}

	// FK
	for _, ref := range refs {
		if fk := convertFK(e, &ref, entityByID, colByID); fk != nil {
			t.FKs = append(t.FKs, *fk)
		}
	}

	return t
}

func convertColumn(col *PDDColumn, domainNames map[string]string) pgd.Column {
	decoded := decodeEscape(col.Type)
	// Check if type is a domain reference (quoted name like "Name" after decode)
	if dn, ok := domainNames[decoded]; ok {
		decoded = dn
	}
	typeName := pgd.NormalizeType(decoded)
	c := pgd.Column{Name: col.Name, Type: typeName}

	if col.Width > 0 && pgd.NeedsLength(typeName) {
		c.Length = col.Width
	}
	if col.Width > 0 && isNumeric(typeName) {
		c.Precision = col.Width
		if col.Prec > 0 {
			c.Scale = col.Prec
		}
	}
	if col.NotNull == 1 {
		c.Nullable = "false"
	}
	if col.Collation != "" {
		c.Collation = col.Collation
	}

	def := decodeEscape(col.DefaultValue)
	if def != "" && !strings.Contains(strings.ToLower(def), "nextval(") {
		if col.QuoteDefault == 1 {
			def = "'" + def + "'"
		}
		c.Default = def
	}

	if col.Comments != "" {
		c.Comment = decodeEscape(col.Comments)
	}

	if col.AutoInc > 0 {
		gen := "by-default"
		if col.AutoInc == 2 {
			gen = "always"
		}
		c.Identity = &pgd.Identity{Generated: gen}
	}

	return c
}

func convertFK(e *PDDEntity, ref *PDDReference, entityByID map[int]*PDDEntity, colByID map[int]*PDDColumn) *pgd.ForeignKey {
	src := entityByID[ref.Source]
	if src == nil {
		return nil
	}
	pairs := parseFKIDS(ref.FKIDS)
	if len(pairs) == 0 {
		return nil
	}

	name := ref.Name
	if name == "" {
		fkCol := ""
		if c := colByID[pairs[0].fk]; c != nil {
			fkCol = c.Name
		}
		name = fmt.Sprintf("fk_%s_%s_%s", e.Name, fkCol, src.Name)
	}

	fk := &pgd.ForeignKey{
		Name: name, ToTable: src.Name,
		OnDelete: refAction(ref.OnDelete), OnUpdate: refAction(ref.OnUpdate),
	}
	if ref.Deferrable == 1 {
		fk.Deferrable = "true"
	}
	for _, p := range pairs {
		// Resolve within the specific entity first (PDD may have duplicate column IDs across tables).
		pk := findColInEntity(src, p.pk)
		if pk == nil {
			pk = colByID[p.pk]
		}
		fkc := findColInEntity(e, p.fk)
		if fkc == nil {
			fkc = colByID[p.fk]
		}
		if pk == nil || fkc == nil {
			continue
		}
		fk.Columns = append(fk.Columns, pgd.FKCol{Name: fkc.Name, References: pk.Name})
	}
	return fk
}

func findColInEntity(e *PDDEntity, id int) *PDDColumn {
	for i := range e.Columns {
		if e.Columns[i].ID == id {
			return &e.Columns[i]
		}
	}
	return nil
}

func convertIndex(tableName string, idx *PDDIndex) *pgd.Index {
	if idx.Name == "" {
		return nil
	}
	cols := parseCommaText(decodeEscape(idx.RawCols.Text))
	if len(cols) == 0 {
		return nil
	}

	out := &pgd.Index{Name: idx.Name, Table: tableName}
	if idx.Unique == 1 {
		out.Unique = "true"
	}
	if m := indexMethod(idx.Method); m != "btree" {
		out.Using = m
	}

	sorts := parseCommaText(decodeEscape(idx.RawSorts.Text))
	nulls := parseCommaText(decodeEscape(idx.RawNulls.Text))

	for i, col := range cols {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}
		if pgd.IsExpression(col) {
			out.Expressions = append(out.Expressions, pgd.Expression{Value: col})
			continue
		}
		name := strings.Trim(col, `"`)
		ref := pgd.ColRef{Name: name}
		if i < len(sorts) && strings.ToLower(sorts[i]) == "desc" {
			ref.Order = "desc"
		}
		if i < len(nulls) {
			n := strings.ToLower(nulls[i])
			switch {
			case strings.Contains(n, "first"):
				ref.Nulls = "first"
			case strings.Contains(n, "last"):
				ref.Nulls = "last"
			}
		}
		out.Columns = append(out.Columns, ref)
	}

	if pred := decodeEscape(idx.Predicate); pred != "" {
		out.Where = &pgd.WhereClause{Value: pred}
	}
	return out
}

// PDD helpers

func sortedColumns(cols []PDDColumn) []PDDColumn {
	out := make([]PDDColumn, len(cols))
	copy(out, cols)
	sort.Slice(out, func(i, j int) bool { return out[i].Pos < out[j].Pos })
	return out
}

func constraintCols(e *PDDEntity, kind int, colByID map[int]*PDDColumn) []string {
	for _, con := range e.Constraints {
		if con.Kind == kind {
			return resolveColNames(&con, colByID)
		}
	}
	if kind == 2 {
		var names []string
		for _, col := range sortedColumns(e.Columns) {
			if col.PrimaryKey == 1 {
				names = append(names, col.Name)
			}
		}
		return names
	}
	return nil
}

func constraintName(e *PDDEntity, kind int) string {
	for _, con := range e.Constraints {
		if con.Kind == kind && con.Name != "" {
			return con.Name
		}
	}
	return ""
}

func resolveColNames(con *PDDConstraint, colByID map[int]*PDDColumn) []string {
	var names []string
	for _, part := range parseCommaText(con.RawCols.Text) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if id, err := strconv.Atoi(part); err == nil {
			if col := colByID[id]; col != nil {
				names = append(names, col.Name)
				continue
			}
		}
		names = append(names, part)
	}
	return names
}

// decodeEscape converts PDD proprietary escaping: \a=' \A=" \g=> \k=< \n=newline
func decodeEscape(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '\\' {
			switch s[i+1] {
			case 'a':
				b.WriteByte('\'')
			case 'A':
				b.WriteByte('"')
			case 'g':
				b.WriteByte('>')
			case 'k':
				b.WriteByte('<')
			case 'n':
				b.WriteByte('\n')
			default:
				b.WriteByte(s[i])
				continue
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isNumeric(t string) bool {
	t = strings.ToLower(t)
	return t == "numeric" || t == "decimal"
}

func indexMethod(m int) string {
	switch m {
	case 2:
		return "hash"
	case 4:
		return "gist"
	case 5:
		return "gin"
	case 6:
		return "brin"
	case 7:
		return "spgist"
	default:
		return "btree"
	}
}

func refAction(code int) string {
	switch code {
	case 1:
		return "cascade"
	case 2:
		return "set-null"
	case 3:
		return "restrict"
	case 4:
		return "set-default"
	default:
		return "no action"
	}
}

func fillColorHex(c int) string {
	if c <= 0 {
		return "#FFFFFF"
	}
	return fmt.Sprintf("#%02X%02X%02X", c&0xFF, (c>>8)&0xFF, (c>>16)&0xFF)
}

type fkIDPair struct{ pk, fk int }

func parseFKIDS(s string) []fkIDPair {
	s = strings.ReplaceAll(s, "\\n", "\n")
	var pairs []fkIDPair
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		pk, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		fk, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if e1 != nil || e2 != nil {
			continue
		}
		pairs = append(pairs, fkIDPair{pk: pk, fk: fk})
	}
	return pairs
}

// parseCommaText parses Delphi COMMATEXT format (quoted strings with double-quote delimiter).
// Doubled quotes inside a quoted string ("") represent a literal double-quote.
func parseCommaText(s string) []string { //nolint:gocognit,nestif,cyclop // COMMATEXT state machine
	if s == "" {
		return nil
	}
	var result []string
	for i := 0; i < len(s); {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' { //nolint:nestif // COMMATEXT quoted string with escaped quotes
			i++ // skip opening quote
			var b strings.Builder
			for i < len(s) {
				if s[i] == '"' {
					if i+1 < len(s) && s[i+1] == '"' {
						b.WriteByte('"')
						i += 2
						continue
					}
					break // closing quote
				}
				b.WriteByte(s[i])
				i++
			}
			result = append(result, b.String())
			if i < len(s) {
				i++ // closing quote
			}
			if i < len(s) && s[i] == ',' {
				i++
			}
		} else {
			start := i
			for i < len(s) && s[i] != ',' {
				i++
			}
			result = append(result, strings.TrimSpace(s[start:i]))
			if i < len(s) {
				i++
			}
		}
	}
	return result
}

// fixPDDXML fixes malformed PDD where self-closing <REFERENCE .../> has child elements.
func fixPDDXML(data []byte) []byte {
	re := regexp.MustCompile(`(<REFERENCE\s[^>]*?)\s*/>(\s*\n\s*<[A-Z])`)
	return re.ReplaceAll(data, []byte(`$1>$2`))
}

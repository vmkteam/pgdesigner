// Package dm2 converts Toad Data Modeler .dm2 binary files to pgd.Project.
package dm2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Convert parses a Toad Data Modeler .dm2 binary file and returns a pgd.Project.
func Convert(data []byte, name string) (*pgd.Project, error) {
	p, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("dm2: %w", err)
	}
	return convert(p, name), nil
}

// record tags
const (
	tagTable    = 0x67
	tagColumn   = 0x68
	tagIndex    = 0x69
	tagIdxCol   = 0x6a
	tagRelation = 0x6b
)

// column field tags
const (
	fldTableID     = 0x03e8
	fldIsPK        = 0x03e9
	fldTypeCode    = 0x03ea
	fldVarcharLen  = 0x03f2
	fldNotNull     = 0x03f4
	fldRenamedName = 0x03f6 // actual column name if different from dict name
	fldDisplayName = 0x03f7
	fldDefault     = 0x03f8
	fldFKDictID    = 0x03ec // dictionary attribute ID of referenced PK column
	fldFKColID     = 0x03ed // internal ID of FK relationship column
	fldFKTableID   = 0x03ee // referenced parent table ID
	fldProperties  = 0x0404
)

// type codes
const (
	tcBoolean     = 50
	tcVarchar     = 80
	tcDate        = 120
	tcInteger     = 150
	tcInterval    = 160
	tcNumeric     = 215
	tcReal        = 250
	tcSerial      = 270
	tcText        = 280
	tcTime        = 290
	tcTimestamp   = 310
	tcTimestampTZ = 320
	tcUserType    = 340
)

var typeNames = map[int]string{
	tcBoolean:     "boolean",
	tcVarchar:     "varchar",
	tcDate:        "date",
	tcInteger:     "integer",
	tcInterval:    "interval",
	tcNumeric:     "numeric",
	tcReal:        "real",
	tcSerial:      "serial",
	tcText:        "text",
	tcTime:        "time",
	tcTimestamp:   "timestamp",
	tcTimestampTZ: "timestamp with time zone",
	tcUserType:    "text", // hstore/json/jsonb/ltree → fallback to text
}

// tag value sizes: positive = fixed bytes, -1 = length-prefixed string, -2 = properties block
var tagSizes = map[uint16]int{
	0x03e8: 4, 0x03e9: 1, 0x03ea: 4, 0x03eb: 4, 0x03ec: 4,
	0x03ed: 4, 0x03ee: 4, 0x03f1: 1, 0x03f2: 4, 0x03f3: 4,
	0x03f4: 1, 0x03f5: 1, 0x0405: 5, 0x03f6: -1, 0x03f7: -1,
	0x03f8: -1, 0x0406: 5, 0x03f9: 5, 0x03fa: 5, 0x03ff: 5,
	0x0400: 5, 0x040d: 5, 0x03fb: 1, 0x03fc: 5, 0x03fd: 5,
	0x040c: 5, 0x03fe: 1, 0x0401: 5, 0x0402: 4, 0x0403: 4,
	0x0407: 1, 0x0408: 1, 0x0409: 5, 0x040a: 5, 0x040b: 5,
	0x040e: 1, 0x040f: 1, 0x0404: -2,
}

// parsed intermediate types

type dm2File struct {
	tables    []dm2Table
	columns   []dm2Column
	relations []dm2Relation
	indexes   []dm2Index
	idxCols   []dm2IdxCol
	genFKIdx  bool // lGenIndexFk flag
}

type dm2Table struct {
	id   int
	name string
	x, y int
}

type dm2Column struct {
	tableID    int
	name       string // Y field (dictionary name)
	renamed    string // 0x03f6 (actual DDL name, may differ from dict name)
	display    string // 0x03f7
	typeCode   int
	varcharLen int
	notNull    bool
	isPK       bool
	defVal     string
	isArray    bool
	fkTableID  int // 0x03ee: referenced parent table ID (0 = not FK)
	fkColID    int // 0x03ed: referenced column internal ID
	fkDictID   int // 0x03ec: dictionary attribute ID of referenced column
}

type dm2Relation struct {
	name     string
	parentID int // source table id
	childID  int // destination table id
}

type dm2Index struct {
	id      int
	name    string
	tableID int
	unique  bool
	method  string // btree, gist, gin, hash, etc.
}

type dm2IdxCol struct {
	indexID int
	name    string
}

type tableData struct {
	table   *dm2Table
	columns []dm2Column
	fks     []dm2Relation
}

// effectiveName returns the column name to use in DDL.
func (c *dm2Column) effectiveName() string {
	if c.renamed != "" {
		return c.renamed
	}
	if c.display != "" {
		return c.display
	}
	return c.name
}

func (c *dm2Column) typeName() string {
	if n, ok := typeNames[c.typeCode]; ok {
		return n
	}
	return "text"
}

// binary parser

func parse(data []byte) (*dm2File, error) {
	f := &dm2File{}

	for _, tag := range []byte{tagTable, tagColumn, tagRelation, tagIndex, tagIdxCol} {
		pattern := []byte{0xf5, 0x01, tag, 0x00}
		pos := 0
		for {
			p := indexOf(data, pattern, pos)
			if p < 0 {
				break
			}
			switch tag {
			case tagTable:
				if t, ok := parseTable(data, p); ok {
					f.tables = append(f.tables, t)
				}
			case tagColumn:
				if c, ok := parseColumn(data, p); ok {
					f.columns = append(f.columns, c)
				}
			case tagRelation:
				if r, ok := parseRelation(data, p); ok {
					f.relations = append(f.relations, r)
				}
			case tagIndex:
				if idx, ok := parseIndex(data, p); ok {
					f.indexes = append(f.indexes, idx)
				}
			case tagIdxCol:
				if ic, ok := parseIdxCol(data, p); ok {
					f.idxCols = append(f.idxCols, ic)
				}
			}
			pos = p + 1
		}
	}

	// check lGenIndexFk flag in global settings
	if indexOf(data, []byte("lGenIndexFk -1"), 0) >= 0 {
		f.genFKIdx = true
	}

	if len(f.tables) == 0 {
		return nil, errors.New("no tables found")
	}
	return f, nil
}

func parseTable(data []byte, start int) (dm2Table, bool) {
	i := start + 4 // skip f5 01 67 00

	id, i, ok := readXField(data, i)
	if !ok {
		return dm2Table{}, false
	}
	name, i, ok := readYField(data, i)
	if !ok {
		return dm2Table{}, false
	}
	i, ok = skipZField(data, i)
	if !ok {
		return dm2Table{}, false
	}

	// Table tags differ from column tags:
	// e8 03 = display name (length-prefixed string)
	// e9 03 = position: uint16(x) + uint16(pad) + uint16(y) + uint16(pad)
	var x, y int

	// e8: display name string
	if i+2 <= len(data) && binary.LittleEndian.Uint16(data[i:]) == 0x03e8 {
		i += 2
		if i+4 <= len(data) {
			slen := int(binary.LittleEndian.Uint32(data[i:]))
			i += 4 + slen
		}
	}

	// e9: position block (8 bytes: x:u16 + pad:u16 + y:u16 + pad:u16)
	if i+2 <= len(data) && binary.LittleEndian.Uint16(data[i:]) == 0x03e9 {
		i += 2
		if i+8 <= len(data) {
			x = int(binary.LittleEndian.Uint16(data[i:]))
			y = int(binary.LittleEndian.Uint16(data[i+4:]))
		}
	}

	return dm2Table{id: id, name: name, x: x, y: y}, true
}

func parseColumn(data []byte, start int) (dm2Column, bool) {
	i := start + 4

	_, i, ok := readXField(data, i) // column internal id (not needed)
	if !ok {
		return dm2Column{}, false
	}
	name, i, ok := readYField(data, i)
	if !ok {
		return dm2Column{}, false
	}
	i, ok = skipZField(data, i)
	if !ok {
		return dm2Column{}, false
	}

	fields := readTags(data, i)

	c := dm2Column{name: name}

	if v, ok := fields.uint32(fldTableID); ok {
		c.tableID = int(v)
	}
	if v, ok := fields.byte1(fldIsPK); ok {
		c.isPK = v == 1
	}
	if v, ok := fields.uint32(fldTypeCode); ok {
		c.typeCode = int(v)
	}
	if v, ok := fields.uint32(fldVarcharLen); ok {
		c.varcharLen = int(v)
	}
	if v, ok := fields.byte1(fldNotNull); ok {
		c.notNull = v == 1
	}
	if v, ok := fields.str(fldRenamedName); ok && v != "" {
		c.renamed = v
	}
	if v, ok := fields.str(fldDisplayName); ok && v != "" {
		c.display = v
	}
	if v, ok := fields.str(fldDefault); ok && v != "" {
		c.defVal = v
	}
	if props, ok := fields.props(fldProperties); ok {
		c.isArray = props["lAttrIsArray"] != "0"
	}
	if v, ok := fields.uint32(fldFKTableID); ok {
		c.fkTableID = int(v)
	}
	if v, ok := fields.uint32(fldFKColID); ok {
		c.fkColID = int(v)
	}
	if v, ok := fields.uint32(fldFKDictID); ok {
		c.fkDictID = int(v)
	}

	return c, true
}

func parseRelation(data []byte, start int) (dm2Relation, bool) {
	i := start + 4

	_, i, ok := readXField(data, i)
	if !ok {
		return dm2Relation{}, false
	}
	name, i, ok := readYField(data, i)
	if !ok {
		return dm2Relation{}, false
	}
	i, ok = skipZField(data, i)
	if !ok {
		return dm2Relation{}, false
	}

	fields := readTags(data, i)

	var parentID, childID int
	if v, ok := fields.uint32(fldTableID); ok {
		parentID = int(v)
	}
	if v, ok := fields.byte1(fldIsPK); ok {
		childID = int(v)
	}

	return dm2Relation{name: name, parentID: parentID, childID: childID}, true
}

// Index tag sizes differ from column tags.
var idxTagSizes = map[uint16]int{
	0x03e8: 4, 0x03e9: 1, 0x03ea: 1, 0x03eb: 5,
	0x03ec: 5, 0x03ed: 1, 0x03ee: 1, 0x03ef: 1,
	0x03f0: 4, 0x03f2: 5, 0x03f3: 5, 0x03f1: -2,
}

func parseIndex(data []byte, start int) (dm2Index, bool) {
	i := start + 4

	id, i, ok := readXField(data, i)
	if !ok {
		return dm2Index{}, false
	}
	name, i, ok := readYField(data, i)
	if !ok {
		return dm2Index{}, false
	}
	i, ok = skipZField(data, i)
	if !ok {
		return dm2Index{}, false
	}

	fields := readTagsCustom(data, i, idxTagSizes)

	idx := dm2Index{id: id, name: name, method: "btree"}
	if v, ok := fields.uint32(fldTableID); ok {
		idx.tableID = int(v)
	}
	if v, ok := fields.byte1(fldIsPK); ok {
		idx.unique = v == 1
	}
	if props, ok := fields.props(0x03f1); ok {
		if m, ok := props["eIxAccess"]; ok && m != "" {
			idx.method = m
		}
	}

	return idx, true
}

func parseIdxCol(data []byte, start int) (dm2IdxCol, bool) {
	i := start + 4

	_, i, ok := readXField(data, i)
	if !ok {
		return dm2IdxCol{}, false
	}
	name, i, ok := readYField(data, i)
	if !ok {
		return dm2IdxCol{}, false
	}
	i, ok = skipZField(data, i)
	if !ok {
		return dm2IdxCol{}, false
	}

	// e8 = parent index ID
	var indexID int
	if i+6 <= len(data) && binary.LittleEndian.Uint16(data[i:]) == 0x03e8 {
		indexID = int(binary.LittleEndian.Uint32(data[i+2:]))
	}

	return dm2IdxCol{indexID: indexID, name: name}, true
}

// readXField reads: 0x58 type(1) uint32_value(4). Returns value and new offset.
func readXField(data []byte, i int) (int, int, bool) {
	if i+6 > len(data) || data[i] != 0x58 {
		return 0, i, false
	}
	v := int(binary.LittleEndian.Uint32(data[i+2:]))
	return v, i + 6, true
}

// readYField reads: 0x59 type(1) len(4) string(len). Returns string and new offset.
func readYField(data []byte, i int) (string, int, bool) {
	if i+6 > len(data) || data[i] != 0x59 {
		return "", i, false
	}
	slen := int(binary.LittleEndian.Uint32(data[i+2:]))
	end := i + 6 + slen
	if end > len(data) {
		return "", i, false
	}
	s := strings.TrimRight(string(data[i+6:end]), "\x00")
	return s, end, true
}

// skipZField skips: 0x5a type(1) guid(16).
func skipZField(data []byte, i int) (int, bool) {
	if i+18 > len(data) || data[i] != 0x5a {
		return i, false
	}
	return i + 18, true
}

// tagValues stores parsed tag-value pairs.
type tagValues struct {
	ints    map[uint16]uint32
	bytes   map[uint16]byte
	strings map[uint16]string
	propMap map[uint16]map[string]string
	offsets map[uint16]int // absolute offset of value start
}

func (tv *tagValues) uint32(tag uint16) (uint32, bool) { v, ok := tv.ints[tag]; return v, ok }
func (tv *tagValues) byte1(tag uint16) (byte, bool)    { v, ok := tv.bytes[tag]; return v, ok }
func (tv *tagValues) str(tag uint16) (string, bool)    { v, ok := tv.strings[tag]; return v, ok }

func (tv *tagValues) props(tag uint16) (map[string]string, bool) {
	v, ok := tv.propMap[tag]
	return v, ok
}

func readTagsCustom(data []byte, i int, sizes map[uint16]int) tagValues {
	return readTagsWithSizes(data, i, sizes)
}

func readTags(data []byte, i int) tagValues {
	return readTagsWithSizes(data, i, tagSizes)
}

func readTagsWithSizes(data []byte, i int, sizes map[uint16]int) tagValues {
	tv := tagValues{
		ints:    make(map[uint16]uint32),
		bytes:   make(map[uint16]byte),
		strings: make(map[uint16]string),
		propMap: make(map[uint16]map[string]string),
		offsets: make(map[uint16]int),
	}

	for i+2 <= len(data) {
		tag := binary.LittleEndian.Uint16(data[i:])
		sz, ok := sizes[tag]
		if !ok {
			break
		}
		i += 2
		tv.offsets[tag] = i

		switch sz {
		case -1: // string
			if i+4 > len(data) {
				return tv
			}
			slen := int(binary.LittleEndian.Uint32(data[i:]))
			i += 4
			if i+slen > len(data) {
				return tv
			}
			tv.strings[tag] = strings.TrimRight(string(data[i:i+slen]), "\x00")
			i += slen

		case -2: // properties block
			if i+4 > len(data) {
				return tv
			}
			count := int(binary.LittleEndian.Uint32(data[i:]))
			i += 4
			props := make(map[string]string, count)
			for range count {
				if i+4 > len(data) {
					return tv
				}
				plen := int(binary.LittleEndian.Uint32(data[i:]))
				i += 4
				if i+plen > len(data) {
					return tv
				}
				pv := strings.TrimRight(string(data[i:i+plen]), "\x00")
				i += plen
				if k, v, ok := strings.Cut(pv, " "); ok {
					props[k] = v
				}
			}
			tv.propMap[tag] = props

		case 1:
			if i >= len(data) {
				return tv
			}
			tv.bytes[tag] = data[i]
			i++

		default: // 4 or 5 byte fixed
			if i+sz > len(data) {
				return tv
			}
			tv.ints[tag] = binary.LittleEndian.Uint32(data[i:])
			i += sz
		}
	}
	return tv
}

func indexOf(data, pattern []byte, from int) int {
	pLen := len(pattern)
	for i := from; i <= len(data)-pLen; i++ {
		match := true
		for j := range pLen {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// converter: dm2File → pgd.Project

func convert(f *dm2File, projectName string) *pgd.Project {
	tableByID := make(map[int]*dm2Table, len(f.tables))
	for i := range f.tables {
		tableByID[f.tables[i].id] = &f.tables[i]
	}

	// group columns by table
	tablesMap := make(map[int]*tableData)
	for _, t := range f.tables {
		tablesMap[t.id] = &tableData{table: &f.tables[0]}
	}
	for i := range f.tables {
		tablesMap[f.tables[i].id].table = &f.tables[i]
	}
	for _, c := range f.columns {
		if td, ok := tablesMap[c.tableID]; ok {
			td.columns = append(td.columns, c)
		}
	}
	for _, r := range f.relations {
		if td, ok := tablesMap[r.childID]; ok {
			td.fks = append(td.fks, r)
		}
	}

	schema := pgd.Schema{Name: "public"}
	var entities []pgd.LayoutEntity

	// build PK map and columns map for FK inference
	pkByTable := make(map[int][]string)
	colsByTable := make(map[int][]dm2Column)
	for tid, td := range tablesMap {
		colsByTable[tid] = td.columns
		for _, c := range td.columns {
			if c.isPK {
				pkByTable[tid] = append(pkByTable[tid], c.effectiveName())
			}
		}
	}

	// maintain file order
	for _, t := range f.tables {
		td := tablesMap[t.id]
		tbl := convertTable(td, tableByID, pkByTable, colsByTable)
		schema.Tables = append(schema.Tables, tbl)
		entities = append(entities, pgd.LayoutEntity{
			Schema: "public", Table: t.name,
			X: t.x, Y: t.y,
		})
	}

	// explicit indexes
	idxColsByIdx := make(map[int][]string)
	for _, ic := range f.idxCols {
		idxColsByIdx[ic.indexID] = append(idxColsByIdx[ic.indexID], ic.name)
	}
	for _, idx := range f.indexes {
		tbl := tableByID[idx.tableID]
		if tbl == nil {
			continue
		}
		pidx := pgd.Index{
			Name:  idx.name,
			Table: tbl.name,
		}
		if idx.unique {
			pidx.Unique = "true"
		}
		if idx.method != "" && idx.method != "btree" {
			pidx.Using = idx.method
		}
		for _, colName := range idxColsByIdx[idx.id] {
			pidx.Columns = append(pidx.Columns, pgd.ColRef{Name: colName})
		}
		schema.Indexes = append(schema.Indexes, pidx)
	}

	// auto-generated FK indexes (lGenIndexFk flag)
	if f.genFKIdx {
		for _, tbl := range schema.Tables {
			for _, fk := range tbl.FKs {
				if len(fk.Columns) == 0 {
					continue
				}
				idx := pgd.Index{
					Name:  "IX_" + fk.Name + "_" + tbl.Name,
					Table: tbl.Name,
				}
				for _, c := range fk.Columns {
					idx.Columns = append(idx.Columns, pgd.ColRef{Name: c.Name})
				}
				schema.Indexes = append(schema.Indexes, idx)
			}
		}
	}

	return &pgd.Project{
		Version: 1, PgVersion: "18", DefaultSchema: "public",
		ProjectMeta: pgd.ProjectMeta{
			Name: projectName,
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

func convertTable(td *tableData, tableByID map[int]*dm2Table, pkByTable map[int][]string, colsByTable map[int][]dm2Column) pgd.Table {
	tbl := pgd.Table{Name: td.table.name}

	var pkCols []string
	for _, c := range td.columns {
		col := convertColumn(&c)
		tbl.Columns = append(tbl.Columns, col)
		if c.isPK {
			pkCols = append(pkCols, col.Name)
		}
	}

	if len(pkCols) > 0 {
		refs := make([]pgd.ColRef, len(pkCols))
		for i, n := range pkCols {
			refs[i] = pgd.ColRef{Name: n}
		}
		tbl.PK = &pgd.PrimaryKey{Columns: refs}
	}

	// FK: use relation records for names + column-level fkTableID for column mapping.
	// Build a map: parentTableID → []fkColumnInChild for this table's columns.
	type fkColInfo struct {
		childCol  string // column name in child table
		parentCol string // PK column name in parent table
	}
	fkColsByParent := make(map[int][]fkColInfo)
	for _, c := range td.columns {
		if c.fkTableID == 0 {
			continue
		}
		childName := c.effectiveName()
		// find the referenced PK column name: it's the dict name (Y field) of the column
		// whose internal ID matches fkDictID in the parent table
		parentColName := findPKColName(c.fkDictID, colsByTable[c.fkTableID])
		if parentColName == "" {
			// fallback: use parent's first PK column
			if pk := pkByTable[c.fkTableID]; len(pk) > 0 {
				parentColName = pk[0]
			}
		}
		fkColsByParent[c.fkTableID] = append(fkColsByParent[c.fkTableID], fkColInfo{
			childCol: childName, parentCol: parentColName,
		})
	}

	seen := make(map[string]bool)
	for _, r := range td.fks {
		parent := tableByID[r.parentID]
		if parent == nil {
			continue
		}
		if seen[r.name] {
			continue
		}
		seen[r.name] = true

		fk := pgd.ForeignKey{
			Name:     r.name,
			ToTable:  parent.name,
			OnDelete: "restrict",
			OnUpdate: "restrict",
		}

		// match FK columns: each relation points to a parent table,
		// find child columns that reference that parent
		if cols, ok := fkColsByParent[r.parentID]; ok {
			// for multi-column FKs pointing to same parent, each relation
			// consumes one column mapping (in order)
			if len(cols) > 0 {
				fk.Columns = append(fk.Columns, pgd.FKCol{
					Name:       cols[0].childCol,
					References: cols[0].parentCol,
				})
				// remove consumed column
				fkColsByParent[r.parentID] = cols[1:]
			}
		}

		tbl.FKs = append(tbl.FKs, fk)
	}

	return tbl
}

// findPKColName finds the effective column name by its dictionary attribute ID.
func findPKColName(dictID int, cols []dm2Column) string {
	for _, c := range cols {
		if c.isPK && c.fkDictID == dictID {
			return c.effectiveName()
		}
	}
	// also check by column's own fkDictID matching
	for _, c := range cols {
		if c.isPK {
			return c.effectiveName()
		}
	}
	return ""
}

func convertColumn(c *dm2Column) pgd.Column {
	typeName := c.typeName()
	col := pgd.Column{
		Name: c.effectiveName(),
		Type: typeName,
	}

	if typeName == "varchar" && c.varcharLen > 0 {
		col.Length = c.varcharLen
	}
	if typeName == "numeric" && c.varcharLen > 0 {
		col.Precision = c.varcharLen
	}

	if c.notNull {
		col.Nullable = "false"
	}

	if c.defVal != "" {
		col.Default = c.defVal
	}

	if c.isArray {
		col.Type = typeName + "[]"
	}

	// serial → integer + identity
	if c.typeCode == tcSerial {
		col.Type = "integer"
		col.Identity = &pgd.Identity{Generated: "by-default"}
	}

	return col
}

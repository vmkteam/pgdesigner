package pgd

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// ERDSchema is the top-level structure for the ERD viewer.
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

// ERDIndex represents an index/constraint shown in the ERD table card.
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

// ToERDSchema converts a Project to an ERDSchema for the ERD viewer.
func (p *Project) ToERDSchema() ERDSchema { //nolint:gocognit // multi-schema support adds complexity
	var schema ERDSchema

	// erdKey returns a unique table key: "table" for public, "schema.table" otherwise.
	defaultSchema := p.DefaultSchema
	if defaultSchema == "" {
		defaultSchema = "public"
	}
	erdKey := func(schemaName, tableName string) string {
		if schemaName == defaultSchema {
			return tableName
		}
		return schemaName + "." + tableName
	}

	// Resolve FK target to erdKey (ToTable may be "table" or "schema.table")
	erdFKTarget := func(fromSchema, toTable string) string {
		// If already qualified, use as-is but apply erdKey logic
		for _, s := range p.Schemas {
			if s.Name+"."+toTable == toTable {
				return toTable // already qualified
			}
		}
		// Try same schema first
		for _, t := range p.Schemas {
			if t.Name == fromSchema {
				for _, tbl := range t.Tables {
					if tbl.Name == toTable {
						return erdKey(fromSchema, toTable)
					}
				}
			}
		}
		// Try default schema
		for _, t := range p.Schemas {
			if t.Name == defaultSchema {
				for _, tbl := range t.Tables {
					if tbl.Name == toTable {
						return erdKey(defaultSchema, toTable)
					}
				}
			}
		}
		return toTable
	}

	// Build PK and FK column sets per table
	pkCols := map[string]map[string]bool{}
	fkCols := map[string]map[string]bool{}

	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			key := erdKey(s.Name, t.Name)
			pk := map[string]bool{}
			if t.PK != nil {
				for _, c := range t.PK.Columns {
					pk[c.Name] = true
				}
			}
			pkCols[key] = pk

			fk := map[string]bool{}
			for _, f := range t.FKs {
				for _, c := range f.Columns {
					fk[c.Name] = true
					schema.References = append(schema.References, ERDReference{
						Name:    f.Name,
						From:    key,
						FromCol: c.Name,
						To:      erdFKTarget(s.Name, f.ToTable),
						ToCol:   c.References,
					})
				}
			}
			fkCols[key] = fk
		}
	}

	// Build tables
	for _, s := range p.Schemas {
		for _, t := range s.Tables {
			key := erdKey(s.Name, t.Name)
			et := ERDTable{Name: key, Schema: s.Name}

			for _, c := range t.Columns {
				ec := ERDColumn{
					Name: c.Name,
					Type: formatColumnType(c),
					PK:   pkCols[key][c.Name],
					NN:   c.Nullable == "false",
					FK:   fkCols[key][c.Name],
				}
				if c.Default != "" {
					ec.Default = shortDefault(c.Default)
				}
				et.Columns = append(et.Columns, ec)
			}

			for _, idx := range s.Indexes {
				if idx.Table == t.Name {
					et.Indexes = append(et.Indexes, ERDIndex{Name: idx.Name})
				}
			}
			for _, u := range t.Uniques {
				et.Indexes = append(et.Indexes, ERDIndex{Name: u.Name})
			}
			for _, ch := range t.Checks {
				et.Indexes = append(et.Indexes, ERDIndex{Name: ch.Name})
			}

			if t.PartitionBy != nil {
				et.Partitioned = true
				et.PartitionCount = len(t.Partitions)
			}

			schema.Tables = append(schema.Tables, et)
		}
	}

	// Assign positions from layout or grid fallback
	assignERDPositions(p, schema.Tables)

	return schema
}

// ToJSSchema converts a Project to a JavaScript const schema = {...} string.
func (p *Project) ToJSSchema() string {
	schema := p.ToERDSchema()

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Sprintf("// error: %v\nconst schema = { tables: [], references: [] };\n", err)
	}

	return fmt.Sprintf("// %d tables, %d references\nconst schema = %s;\n",
		len(schema.Tables), len(schema.References), string(data))
}

func assignERDPositions(p *Project, tables []ERDTable) {
	layoutPos := map[string][2]int{}
	if len(p.Layouts.Layouts) > 0 {
		for _, e := range p.Layouts.Layouts[0].Entities {
			layoutPos[e.Table] = [2]int{e.X, e.Y}
		}
	}

	colsCount := int(math.Ceil(math.Sqrt(float64(len(tables))))) + 1
	if colsCount < 3 {
		colsCount = 3
	}

	for i := range tables {
		if pos, ok := layoutPos[tables[i].Name]; ok {
			tables[i].X = pos[0]
			tables[i].Y = pos[1]
		} else {
			tables[i].X = (i%colsCount)*380 + 40
			tables[i].Y = (i/colsCount)*450 + 40
		}
	}
}

func shortDefault(s string) string {
	// Normalize common defaults for compact display
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "current_timestamp" || low == "now()" {
		return "now()"
	}

	const maxLen = 32
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatColumnType(c Column) string {
	t := c.Type
	switch {
	case c.Length > 0:
		t += fmt.Sprintf("(%d)", c.Length)
	case c.Precision > 0 && c.Scale > 0:
		t += fmt.Sprintf("(%d,%d)", c.Precision, c.Scale)
	case c.Precision > 0:
		t += fmt.Sprintf("(%d)", c.Precision)
	}
	return t
}

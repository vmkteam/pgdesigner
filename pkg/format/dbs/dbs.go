// Package dbs converts DbSchema .dbs XML files to pgd.Project.
package dbs

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

// Convert parses a DbSchema .dbs XML file and returns a pgd.Project.
// The name parameter is ignored (project name is taken from the .dbs file).
func Convert(data []byte, name string) (*pgd.Project, error) {
	var dbs DBS
	if err := xml.Unmarshal(data, &dbs); err != nil {
		return nil, fmt.Errorf("parsing dbs XML: %w", err)
	}
	return convert(&dbs), nil
}

// DBS input structs

type DBS struct {
	XMLName xml.Name    `xml:"project"`
	Name    string      `xml:"name,attr"`
	Schemas []DBSSchema `xml:"schema"`
	Layouts []DBSLayout `xml:"layout"`
}

type DBSSchema struct {
	Name   string     `xml:"name,attr"`
	Tables []DBSTable `xml:"table"`
}

type DBSTable struct {
	Name    string      `xml:"name,attr"`
	Columns []DBSColumn `xml:"column"`
	Indexes []DBSIndex  `xml:"index"`
	FKs     []DBSFK     `xml:"fk"`
}

type DBSColumn struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`
	Length    string `xml:"length,attr"`
	Mandatory string `xml:"mandatory,attr"`
	Identity  string `xml:"identity"`
	Default   string `xml:"defo"`
}

type DBSIndex struct {
	Name    string        `xml:"name,attr"`
	Unique  string        `xml:"unique,attr"`
	Columns []DBSIndexCol `xml:"column"`
}

type DBSIndexCol struct {
	Name string `xml:"name,attr"`
}

type DBSFK struct {
	Name         string     `xml:"name,attr"`
	ToTable      string     `xml:"to_table,attr"`
	DeleteAction string     `xml:"delete_action,attr"`
	UpdateAction string     `xml:"update_action,attr"`
	Columns      []DBSFKCol `xml:"fk_column"`
}

type DBSFKCol struct {
	Name string `xml:"name,attr"`
	PK   string `xml:"pk,attr"`
}

type DBSLayout struct {
	Name     string         `xml:"name,attr"`
	Entities []DBSLayoutEnt `xml:"entity"`
	Groups   []DBSLayoutGrp `xml:"group"`
}

type DBSLayoutEnt struct {
	Schema string `xml:"schema,attr"`
	Name   string `xml:"name,attr"`
	Color  string `xml:"color,attr"`
	X      int    `xml:"x,attr"`
	Y      int    `xml:"y,attr"`
}

type DBSLayoutGrp struct {
	Name     string         `xml:"name,attr"`
	Color    string         `xml:"color,attr"`
	Entities []DBSLayoutEnt `xml:"entity"`
}

func convert(dbs *DBS) *pgd.Project {
	p := &pgd.Project{
		Version:       1,
		PgVersion:     "18",
		DefaultSchema: "public",
		ProjectMeta: pgd.ProjectMeta{
			Name: dbs.Name,
			Settings: pgd.Settings{
				Naming:   pgd.Naming{Convention: "camelCase"},
				Defaults: pgd.Defaults{Nullable: "true", OnDelete: "restrict", OnUpdate: "restrict"},
			},
		},
	}
	for _, s := range dbs.Schemas {
		p.Schemas = append(p.Schemas, convertSchema(&s))
	}
	if len(dbs.Layouts) > 0 {
		p.Layouts = convertLayouts(dbs.Layouts)
	}
	return p
}

func convertSchema(s *DBSSchema) pgd.Schema {
	out := pgd.Schema{Name: s.Name}
	for i := range s.Tables {
		tbl, idxs := convertTable(&s.Tables[i])
		out.Tables = append(out.Tables, tbl)
		out.Indexes = append(out.Indexes, idxs...)
	}
	return out
}

func convertTable(t *DBSTable) (pgd.Table, []pgd.Index) {
	tbl := pgd.Table{Name: t.Name}
	var idxs []pgd.Index

	for _, col := range t.Columns {
		tbl.Columns = append(tbl.Columns, convertColumn(&col))
	}
	for _, idx := range t.Indexes {
		switch idx.Unique {
		case "PRIMARY_KEY":
			tbl.PK = &pgd.PrimaryKey{Name: idx.Name, Columns: colRefs(idx.Columns)}
		case "UNIQUE_INDEX":
			tbl.Uniques = append(tbl.Uniques, pgd.Unique{Name: idx.Name, Columns: colRefs(idx.Columns)})
		default:
			idxs = append(idxs, pgd.Index{Name: idx.Name, Table: t.Name, Columns: colRefs(idx.Columns)})
		}
	}
	for _, fk := range t.FKs {
		tbl.FKs = append(tbl.FKs, convertFK(&fk))
	}
	return tbl, idxs
}

func convertColumn(col *DBSColumn) pgd.Column {
	c := pgd.Column{
		Name: col.Name,
		Type: pgd.NormalizeType(col.Type),
	}
	if l := atoi(col.Length); l > 0 && pgd.NeedsLength(c.Type) {
		c.Length = l
	}
	if col.Mandatory == "y" {
		c.Nullable = "false"
	}
	if def := strings.TrimSpace(col.Default); def != "" {
		c.Default = def
	}
	if id := strings.TrimSpace(col.Identity); id != "" {
		gen := "by-default"
		if strings.Contains(strings.ToUpper(id), "ALWAYS") {
			gen = "always"
		}
		c.Identity = &pgd.Identity{Generated: gen}
	}
	return c
}

func convertFK(fk *DBSFK) pgd.ForeignKey {
	f := pgd.ForeignKey{
		Name:     fk.Name,
		ToTable:  fk.ToTable,
		OnDelete: normalizeAction(fk.DeleteAction),
		OnUpdate: normalizeAction(fk.UpdateAction),
	}
	for _, col := range fk.Columns {
		f.Columns = append(f.Columns, pgd.FKCol{Name: col.Name, References: col.PK})
	}
	return f
}

func convertLayouts(layouts []DBSLayout) pgd.Layouts {
	var out []pgd.Layout
	for i, l := range layouts {
		pl := pgd.Layout{Name: l.Name}
		if i == 0 {
			pl.Default = "true"
		}
		for _, e := range l.Entities {
			pl.Entities = append(pl.Entities, pgd.LayoutEntity{
				Schema: coalesce(e.Schema, "public"),
				Table:  e.Name,
				X:      e.X, Y: e.Y,
				Color: ensureHash(e.Color),
			})
		}
		for _, g := range l.Groups {
			grp := pgd.LayoutGroup{Name: g.Name, Color: ensureHash(g.Color)}
			for _, e := range g.Entities {
				grp.Members = append(grp.Members, pgd.LayoutMember{
					Schema: coalesce(e.Schema, "public"), Table: e.Name,
				})
			}
			pl.Groups = append(pl.Groups, grp)
		}
		out = append(out, pl)
	}
	return pgd.Layouts{Layouts: out}
}

func colRefs(cols []DBSIndexCol) []pgd.ColRef {
	out := make([]pgd.ColRef, len(cols))
	for i, c := range cols {
		out[i] = pgd.ColRef{Name: c.Name}
	}
	return out
}

func normalizeAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "cascade":
		return "cascade"
	case "set null", "set-null", "setnull", "set_null":
		return "set-null"
	case "set default", "set-default", "setdefault", "set_default":
		return "set-default"
	case "no action", "noaction", "no_action":
		return "no action"
	default:
		return "restrict"
	}
}

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func ensureHash(c string) string {
	if c != "" && !strings.HasPrefix(c, "#") {
		return "#" + c
	}
	return c
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

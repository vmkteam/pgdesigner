package rpc

import "github.com/vmkteam/pgdesigner/pkg/pgd"

// --- Input types for UpdateTable ---

// GeneralInput holds editable table properties.
type GeneralInput struct {
	Name     *string `json:"name,omitempty"`
	Comment  *string `json:"comment,omitempty"`
	Unlogged *bool   `json:"unlogged,omitempty"`
	Generate *bool   `json:"generate,omitempty"`
}

// ColumnInput is a column from the editor.
type ColumnInput struct {
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Length          int             `json:"length,omitempty"`
	Precision       int             `json:"precision,omitempty"`
	Scale           int             `json:"scale,omitempty"`
	Nullable        bool            `json:"nullable"`
	Default         string          `json:"default,omitempty"`
	Identity        string          `json:"identity,omitempty"`
	IdentitySeqOpt  *IdentitySeqOpt `json:"identitySeqOpt,omitempty"`
	Generated       string          `json:"generated,omitempty"`
	GeneratedStored bool            `json:"generatedStored,omitempty"`
	Comment         string          `json:"comment,omitempty"`
	Compression     string          `json:"compression,omitempty"`
	Storage         string          `json:"storage,omitempty"`
	Collation       string          `json:"collation,omitempty"`
}

func (c ColumnInput) toPGD() pgd.Column {
	col := pgd.Column{
		Name:        c.Name,
		Type:        c.Type,
		Length:      c.Length,
		Precision:   c.Precision,
		Scale:       c.Scale,
		Default:     c.Default,
		Comment:     c.Comment,
		Compression: c.Compression,
		Storage:     c.Storage,
		Collation:   c.Collation,
	}
	if !c.Nullable {
		col.Nullable = "false"
	}
	if c.Identity != "" && c.Generated == "" {
		id := &pgd.Identity{Generated: c.Identity}
		if c.IdentitySeqOpt != nil {
			id.Sequence = &pgd.IdentitySeqOpt{
				Start: c.IdentitySeqOpt.Start, Increment: c.IdentitySeqOpt.Increment,
				Min: c.IdentitySeqOpt.Min, Max: c.IdentitySeqOpt.Max, Cache: c.IdentitySeqOpt.Cache,
			}
			if c.IdentitySeqOpt.Cycle {
				id.Sequence.Cycle = "true"
			}
		}
		col.Identity = id
	}
	if c.Generated != "" && c.Identity == "" {
		stored := "true"
		if !c.GeneratedStored {
			stored = "false"
		}
		col.Generated = &pgd.Generated{Expression: c.Generated, Stored: stored}
	}
	return col
}

// PKInput is a PK constraint from the editor.
type PKInput struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

func (p *PKInput) toPGD() *pgd.PrimaryKey {
	if p == nil || p.Name == "" {
		return nil
	}
	pk := &pgd.PrimaryKey{Name: p.Name}
	for _, c := range p.Columns {
		pk.Columns = append(pk.Columns, pgd.ColRef{Name: c})
	}
	return pk
}

// FKInput is a FK from the editor.
type FKInput struct {
	Name       string       `json:"name"`
	ToTable    string       `json:"toTable"`
	OnDelete   string       `json:"onDelete"`
	OnUpdate   string       `json:"onUpdate"`
	Deferrable bool         `json:"deferrable,omitempty"`
	Initially  string       `json:"initially,omitempty"`
	Columns    []FKColInput `json:"columns"`
}

// FKColInput maps local → referenced column.
type FKColInput struct {
	Name       string `json:"name"`
	References string `json:"references"`
}

func (f FKInput) toPGD() pgd.ForeignKey {
	fk := pgd.ForeignKey{
		Name:      f.Name,
		ToTable:   f.ToTable,
		OnDelete:  f.OnDelete,
		OnUpdate:  f.OnUpdate,
		Initially: f.Initially,
	}
	if f.Deferrable {
		fk.Deferrable = "true"
	}
	for _, c := range f.Columns {
		fk.Columns = append(fk.Columns, pgd.FKCol{Name: c.Name, References: c.References})
	}
	return fk
}

// UniqueInput is a UNIQUE constraint from the editor.
type UniqueInput struct {
	Name          string   `json:"name"`
	Columns       []string `json:"columns"`
	NullsDistinct bool     `json:"nullsDistinct,omitempty"`
}

func (u UniqueInput) toPGD() pgd.Unique {
	uq := pgd.Unique{Name: u.Name}
	if u.NullsDistinct {
		uq.NullsDistinct = "true"
	}
	for _, c := range u.Columns {
		uq.Columns = append(uq.Columns, pgd.ColRef{Name: c})
	}
	return uq
}

// CheckInput is a CHECK constraint from the editor.
type CheckInput struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

func (c CheckInput) toPGD() pgd.Check {
	return pgd.Check{Name: c.Name, Expression: c.Expression}
}

// ExcludeInput is an EXCLUDE constraint from the editor.
type ExcludeInput struct {
	Name     string                `json:"name"`
	Using    string                `json:"using,omitempty"`
	Elements []ExcludeElementInput `json:"elements"`
	Where    string                `json:"where,omitempty"`
}

// ExcludeElementInput is one element of an exclude constraint.
type ExcludeElementInput struct {
	Column     string `json:"column,omitempty"`
	Expression string `json:"expression,omitempty"`
	With       string `json:"with"`
}

func (e ExcludeInput) toPGD() pgd.Exclude {
	ex := pgd.Exclude{Name: e.Name, Using: e.Using}
	for _, el := range e.Elements {
		ex.Elements = append(ex.Elements, pgd.ExcludeElement{Column: el.Column, Expression: el.Expression, With: el.With})
	}
	if e.Where != "" {
		ex.Where = &pgd.WhereClause{Value: e.Where}
	}
	return ex
}

// IndexInput is an index from the editor.
type IndexInput struct {
	Name          string           `json:"name"`
	Table         string           `json:"table"`
	Unique        bool             `json:"unique,omitempty"`
	NullsDistinct bool             `json:"nullsDistinct,omitempty"`
	Using         string           `json:"using,omitempty"`
	Columns       []IndexColInput  `json:"columns"`
	Expressions   []string         `json:"expressions,omitempty"`
	With          []WithParamInput `json:"with,omitempty"`
	Where         string           `json:"where,omitempty"`
	Include       []string         `json:"include,omitempty"`
}

// WithParamInput is a key-value storage parameter.
type WithParamInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// IndexColInput is an index column with ordering.
type IndexColInput struct {
	Name    string `json:"name"`
	Order   string `json:"order,omitempty"`
	Nulls   string `json:"nulls,omitempty"`
	Opclass string `json:"opclass,omitempty"`
}

func (idx IndexInput) toPGD() pgd.Index {
	ix := pgd.Index{
		Name:  idx.Name,
		Table: idx.Table,
		Using: idx.Using,
	}
	if idx.Unique {
		ix.Unique = "true"
	}
	if idx.NullsDistinct {
		ix.NullsDistinct = "true"
	}
	for _, c := range idx.Columns {
		ix.Columns = append(ix.Columns, pgd.ColRef{Name: c.Name, Order: c.Order, Nulls: c.Nulls, Opclass: c.Opclass})
	}
	for _, e := range idx.Expressions {
		ix.Expressions = append(ix.Expressions, pgd.Expression{Value: e})
	}
	if len(idx.With) > 0 {
		w := &pgd.With{}
		for _, p := range idx.With {
			w.Params = append(w.Params, pgd.WithParam{Name: p.Name, Value: p.Value})
		}
		ix.With = w
	}
	if idx.Where != "" {
		ix.Where = &pgd.WhereClause{Value: idx.Where}
	}
	if len(idx.Include) > 0 {
		inc := &pgd.Include{}
		for _, c := range idx.Include {
			inc.Columns = append(inc.Columns, pgd.ColRef{Name: c})
		}
		ix.Include = inc
	}
	return ix
}

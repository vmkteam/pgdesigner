package rpc

import (
	"github.com/vmkteam/pgdesigner/pkg/designer/diff"
	"github.com/vmkteam/pgdesigner/pkg/designer/lint"
	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

//go:generate colgen -imports=github.com/vmkteam/pgdesigner/pkg/pgd,github.com/vmkteam/pgdesigner/pkg/designer/lint,github.com/vmkteam/pgdesigner/pkg/designer/diff

//colgen:ERDTable:MapP(pgd.ERDTable)
//colgen:ERDColumn:MapP(pgd.ERDColumn)
//colgen:ERDIndex:MapP(pgd.ERDIndex)
//colgen:ERDReference:MapP(pgd.ERDReference)
//colgen:LintIssue:MapP(lint.Issue)
//colgen:DiffChange:MapP(diff.Change)
//colgen:DiffHazard:MapP(diff.Hazard)

// NewERDTable converts pgd.ERDTable to rpc.ERDTable.
func NewERDTable(src *pgd.ERDTable) *ERDTable {
	return &ERDTable{
		Name:           src.Name,
		Schema:         src.Schema,
		X:              src.X,
		Y:              src.Y,
		Columns:        MapP(src.Columns, NewERDColumn),
		Indexes:        MapP(src.Indexes, NewERDIndex),
		Partitioned:    src.Partitioned,
		PartitionCount: src.PartitionCount,
	}
}

// NewERDColumn converts pgd.ERDColumn to rpc.ERDColumn.
func NewERDColumn(src *pgd.ERDColumn) *ERDColumn {
	return &ERDColumn{
		Name:    src.Name,
		Type:    src.Type,
		PK:      src.PK,
		NN:      src.NN,
		FK:      src.FK,
		Default: src.Default,
	}
}

// NewERDIndex converts pgd.ERDIndex to rpc.ERDIndex.
func NewERDIndex(src *pgd.ERDIndex) *ERDIndex {
	return &ERDIndex{Name: src.Name}
}

// NewERDReference converts pgd.ERDReference to rpc.ERDReference.
func NewERDReference(src *pgd.ERDReference) *ERDReference {
	return &ERDReference{
		Name:    src.Name,
		From:    src.From,
		FromCol: src.FromCol,
		To:      src.To,
		ToCol:   src.ToCol,
	}
}

// NewLintIssue converts lint.Issue to rpc.LintIssue.
func NewLintIssue(src *lint.Issue) *LintIssue {
	title := src.Code
	var fixable bool
	if r, ok := lint.Rules[src.Code]; ok {
		title = r.Title
		fixable = r.Fixable
	}
	return &LintIssue{
		Severity: src.Severity.String(),
		Code:     src.Code,
		Title:    title,
		Path:     src.Path,
		Message:  src.Message,
		Fixable:  fixable,
	}
}

// NewDiffChange converts diff.Change to rpc.DiffChange.
func NewDiffChange(src *diff.Change) *DiffChange {
	return &DiffChange{
		Object:  src.Object,
		Action:  src.Action,
		Table:   src.Table,
		Name:    src.Name,
		SQL:     src.SQL,
		Hazards: MapP(src.Hazards, NewDiffHazard),
	}
}

// NewDiffHazard converts diff.Hazard to rpc.DiffHazard.
func NewDiffHazard(src *diff.Hazard) *DiffHazard {
	return &DiffHazard{Level: src.Level, Code: src.Code, Message: src.Message}
}

// MapP converts slice of type T to slice of type M with given converter with pointers.
func MapP[T, M any](a []T, f func(*T) *M) []M {
	n := make([]M, len(a))
	for i := range a {
		n[i] = *f(&a[i])
	}
	return n
}

// MapV converts slice of type T to slice of type M with a value converter.
func MapV[T, M any](a []T, f func(T) M) []M {
	n := make([]M, len(a))
	for i, v := range a {
		n[i] = f(v)
	}
	return n
}

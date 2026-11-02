package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func TestMarkVirtualGenerated(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "virtual keyword",
			in:   `b int GENERATED ALWAYS AS (a * 2) VIRTUAL,`,
			want: `b int GENERATED ALWAYS AS (__pgdesigner_virtual__(a * 2)) STORED,`,
		},
		{
			name: "pg_dump 18 form without keyword",
			in:   "b integer GENERATED ALWAYS AS ((a * 2)),\n c int",
			want: "b integer GENERATED ALWAYS AS (__pgdesigner_virtual__((a * 2))) STORED,\n c int",
		},
		{
			name: "stored is untouched",
			in:   `d text GENERATED ALWAYS AS ((a)::text || ')') STORED`,
			want: `d text GENERATED ALWAYS AS ((a)::text || ')') STORED`,
		},
		{
			name: "identity is untouched",
			in:   `id int GENERATED ALWAYS AS IDENTITY`,
			want: `id int GENERATED ALWAYS AS IDENTITY`,
		},
		{
			name: "lowercase, nested parens and string with paren",
			in:   `x text generated always as (lower(a || ')(')) not null`,
			want: `x text generated always as (__pgdesigner_virtual__(lower(a || ')('))) STORED not null`,
		},
		{
			name: "dollar-quoted body and comments are untouched",
			in:   "$f$ GENERATED ALWAYS AS (1) $f$ /* GENERATED ALWAYS AS (2) */ -- GENERATED ALWAYS AS (3)\n",
			want: "$f$ GENERATED ALWAYS AS (1) $f$ /* GENERATED ALWAYS AS (2) */ -- GENERATED ALWAYS AS (3)\n",
		},
		{
			name: "escape string and nested comment are untouched",
			in:   "E'it\\'s GENERATED ALWAYS AS (1)' /* a /* GENERATED ALWAYS AS (2) */ b */ x",
			want: "E'it\\'s GENERATED ALWAYS AS (1)' /* a /* GENERATED ALWAYS AS (2) */ b */ x",
		},
		{
			name: "comment before STORED",
			in:   `d int GENERATED ALWAYS AS (a) /* c */ STORED`,
			want: `d int GENERATED ALWAYS AS (a) /* c */ STORED`,
		},
		{
			name: "part of identifier",
			in:   `xGENERATED ALWAYS AS (1)`,
			want: `xGENERATED ALWAYS AS (1)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, markVirtualGenerated(tt.in))
		})
	}
}

func TestParseSQL_VirtualGenerated(t *testing.T) {
	// pg_dump 18 output: virtual columns have no VIRTUAL keyword
	sql := `CREATE TABLE public.p (
    a integer,
    b integer GENERATED ALWAYS AS ((a * 2)),
    c integer GENERATED ALWAYS AS (a * 3) VIRTUAL,
    d text GENERATED ALWAYS AS (((a)::text || 'x'::text)) STORED
);`
	p, err := ParseSQL(sql, "test")
	require.NoError(t, err)

	cols := p.Schemas[0].Tables[0].Columns
	require.Len(t, cols, 4)
	assert.Nil(t, cols[0].Generated)
	assert.Equal(t, &pgd.Generated{Expression: "a * 2", Stored: "false"}, cols[1].Generated)
	assert.Equal(t, &pgd.Generated{Expression: "a * 3", Stored: "false"}, cols[2].Generated)
	assert.Equal(t, &pgd.Generated{Expression: "a::text || 'x'::text", Stored: "true"}, cols[3].Generated)

	// round-trip through our own DDL
	p2, err := ParseSQL(pgd.GenerateDDL(p), "test")
	require.NoError(t, err)
	assert.Equal(t, cols, p2.Schemas[0].Tables[0].Columns)
}

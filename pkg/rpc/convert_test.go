package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vmkteam/pgdesigner/pkg/pgd"
)

func TestIndexKeys_RoundTrip(t *testing.T) {
	keys := []pgd.IndexKey{
		{Name: "shopId"},
		{Expression: `COALESCE("updatedAt", "createdAt")`, Order: "desc", Nulls: "last"},
		{Name: "title", Opclass: "text_pattern_ops"},
	}

	// what the editor gets and sends back
	data, err := json.Marshal(MapV(keys, newIndexKeyDetail))
	require.NoError(t, err)
	var input []IndexKeyInput
	require.NoError(t, json.Unmarshal(data, &input))

	assert.Equal(t, keys, MapV(input, IndexKeyInput.toPGD))
	assert.Equal(t, IndexKeyExpression, input[1].Kind)
	assert.Equal(t, IndexKeyColumn, input[2].Kind)
}

func TestNullsNotDistinct_RoundTrip(t *testing.T) {
	// what the editor gets for a table and sends back must keep NULLS NOT DISTINCT
	for _, nullsDistinct := range []string{"", "false"} {
		t.Run("nulls-distinct="+nullsDistinct, func(t *testing.T) {
			idx := pgd.Index{Name: "ix_t_a", Table: "t", Unique: "true", NullsDistinct: nullsDistinct, Keys: []pgd.IndexKey{{Name: "a"}}}
			tbl := pgd.Table{Name: "t", Columns: []pgd.Column{{Name: "a", Type: "int"}},
				Uniques: []pgd.Unique{{Name: "uq_t_a", NullsDistinct: nullsDistinct, Columns: []pgd.ColRef{{Name: "a"}}}}}
			schema := pgd.Schema{Name: "public", Tables: []pgd.Table{tbl}, Indexes: []pgd.Index{idx}}
			p := &pgd.Project{Schemas: []pgd.Schema{schema}}

			data, err := json.Marshal(newTableDetail(p, &schema.Tables[0], &schema))
			require.NoError(t, err)
			var input struct {
				Uniques []UniqueInput `json:"uniques"`
				Indexes []IndexInput  `json:"indexes"`
			}
			require.NoError(t, json.Unmarshal(data, &input))
			require.Len(t, input.Uniques, 1)
			require.Len(t, input.Indexes, 1)

			assert.Equal(t, nullsDistinct, input.Uniques[0].toPGD().NullsDistinct)
			assert.Equal(t, nullsDistinct, input.Indexes[0].toPGD().NullsDistinct)
		})
	}
}

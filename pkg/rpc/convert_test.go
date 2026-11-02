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

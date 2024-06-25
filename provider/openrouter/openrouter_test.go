package openrouter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderModels(t *testing.T) {
	provider := NewProvider()

	models, err := provider.Models()

	require.NoError(t, err)
	assert.NotEmpty(t, models)
}

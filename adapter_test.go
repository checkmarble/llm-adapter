package llmadapter

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	llm, err := New(WithApiKey("apikey"))

	assert.Nil(t, err)
	assert.Equal(t, "apikey", llm.ApiKey())

	llm, err = New(WithDefaultModel("themodel"))

	assert.Nil(t, err)
	assert.Equal(t, "themodel", llm.DefaultModel())

	expectedDefaultProvider, _ := NewMockProvider()
	expectedProvider1, _ := NewMockProvider()
	expectedProvider2, _ := NewMockProvider()

	llm, err = New(
		WithDefaultProvider(expectedDefaultProvider),
		WithProvider("provider1", expectedProvider1),
		WithProvider("provider2", expectedProvider2),
	)

	assert.Nil(t, err)

	defaultProvider, err := llm.GetProvider(nil)

	assert.Nil(t, err)
	assert.Equal(t, expectedDefaultProvider, defaultProvider)

	provider1, err := llm.GetProvider(lo.ToPtr("provider1"))

	assert.Nil(t, err)
	assert.Equal(t, expectedProvider1, provider1)

	provider2, err := llm.GetProvider(lo.ToPtr("provider2"))

	assert.Nil(t, err)
	assert.Equal(t, expectedProvider2, provider2)
}

package llmadapter

import (
	"reflect"
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

type mockProvider1Opts struct {
	Text string
}

func (mockProvider1Opts) RequestOptionsForProvider() {}

type mockProvider1 struct {
	MockProvider
}

func (mockProvider1) RequestOptionsType() reflect.Type {
	return reflect.TypeFor[mockProvider1Opts]()
}

type mockProvider2Opts struct {
	Number int
}

func (mockProvider2Opts) RequestOptionsForProvider() {}

type mockProvider2 struct {
	MockProvider
}

func (mockProvider2) RequestOptionsType() reflect.Type {
	return reflect.TypeFor[mockProvider2Opts]()
}

func TestProviderRequestOptions(t *testing.T) {
	provider1 := mockProvider1{}
	provider2 := mockProvider2{}

	req := NewUntypedRequest().
		WithProviderOptions(mockProvider1Opts{Text: "thetext"}).
		WithProviderOptions(mockProvider2Opts{Number: 42})

	assert.Equal(t, mockProvider1Opts{Text: "thetext"}, req.WithProvider("provider1").ProviderRequestOptions(&provider1))
	assert.Equal(t, mockProvider2Opts{Number: 42}, req.WithProvider("provider2").ProviderRequestOptions(&provider2))
}

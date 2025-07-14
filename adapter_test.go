package llmadapter

import (
	"reflect"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProviderInitError(t *testing.T) {
	p := NewMockProvider()
	p.On("Init", mock.Anything).Return(errors.New("could not initialize provider"))

	llm, err := New(WithDefaultProvider(p))

	assert.ErrorContains(t, err, "could not initialize provider")
	assert.Nil(t, llm)
}

type mockProvider1Opts struct {
	Text string
}

func (mockProvider1Opts) RequestOptionsForProvider() {}

type mockProvider1 struct {
	MockProvider
}

func (*mockProvider1) RequestOptionsType() reflect.Type {
	return reflect.TypeFor[mockProvider1Opts]()
}

type mockProvider2Opts struct {
	Number int
}

func (mockProvider2Opts) RequestOptionsForProvider() {}

type mockProvider2 struct {
	MockProvider
}

func (*mockProvider2) RequestOptionsType() reflect.Type {
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

func TestProviderHistory(t *testing.T) {
	provider1 := NewMockProvider()
	provider2 := NewMockProvider()

	provider1.On("Init", mock.Anything).Return(nil)
	provider2.On("Init", mock.Anything).Return(nil)

	llm, _ := New(
		WithProvider("provider1", provider1),
		WithProvider("provider2", provider2),
		WithSaveContext(),
	)

	provider1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"Hello, world!"}, nil).Once()
	provider2.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"Hello, world!"}, nil)

	resp1, _ := NewUntypedRequest().WithText(RoleUser, "").Do(t.Context(), llm)

	assert.Len(t, provider1.History.Load(), 0)
	assert.Len(t, provider2.History.Load(), 0)

	provider1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"Hello, world 2!"}, nil).Once()

	resp2, _ := NewUntypedRequest().FromCandidate(resp1, 0).Do(t.Context(), llm)

	assert.Len(t, provider1.History.Load(), 1)
	assert.Len(t, provider2.History.Load(), 0)

	provider1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"Hello, world 3!"}, nil).Once()

	_, _ = NewUntypedRequest().FromCandidate(resp2, 0).Do(t.Context(), llm)

	assert.Len(t, provider1.History.Load(), 2)
	assert.ElementsMatch(t, provider1.History.Load(), []MockMessage{{"Hello, world!"}, {"Hello, world 2!"}})
	assert.Len(t, provider2.History.Load(), 0)

	resp4, _ := NewUntypedRequest().WithProvider("provider2").Do(t.Context(), llm)
	cand, _ := resp4.Candidate(0)
	cand.SelectCandidate()

	assert.Len(t, provider1.History.Load(), 2)
	assert.Len(t, provider2.History.Load(), 1)

	llm.ResetContext("provider2")

	assert.Len(t, provider1.History.Load(), 2)
	assert.Len(t, provider2.History.Load(), 0)

	llm.ResetContext()

	assert.Len(t, provider1.History.Load(), 0)
	assert.Len(t, provider2.History.Load(), 0)
}

func TestGetDefaultProvider(t *testing.T) {
	provider := NewMockProvider()
	provider.On("Init", mock.Anything).Return(nil)

	llm, _ := New()
	p, err := llm.GetProvider(nil)

	assert.ErrorContains(t, err, "no provider was configured")
	assert.Nil(t, p)

	llm, _ = New(WithProvider("theprovider", provider))
	p, err = llm.GetProvider(nil)

	assert.Nil(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, provider, p)

	p, err = llm.GetProvider(nil)

	assert.Nil(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, provider, p)

	secondProvider := NewMockProvider()
	secondProvider.On("Init", mock.Anything).Return(nil)

	llm, _ = New(WithDefaultProvider(provider), WithProvider("secondprovider", secondProvider))

	p, err = llm.GetProvider(nil)

	assert.Nil(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, provider, p)
	assert.NotEqual(t, secondProvider, p)

	p, err = llm.GetProvider(lo.ToPtr("secondprovider"))

	assert.Nil(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, secondProvider, p)
	assert.NotEqual(t, provider, p)

	p, err = llm.GetProvider(lo.ToPtr("unknown"))

	assert.ErrorContains(t, err, "unknown provider")
	assert.Nil(t, p)
}

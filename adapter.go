package llmadapter

import (
	"context"
	"net/http"
	"reflect"

	"github.com/checkmarble/marble-llm-adapter/internal"
	"github.com/cockroachdb/errors"
)

const (
	defaultProvider = "__DEFAULT__"
)

type Llm interface {
	Init(llm internal.Adapter) error
	ResetContext()
	ChatCompletion(context.Context, internal.Adapter, Requester) (*InnerResponse, error)
	RequestOptionsType() reflect.Type
}

// LlmAdapter is the main entrypoint for interacting with different LLM providers.
// It provides a unified interface to send requests and receive responses.
type LlmAdapter struct {
	providers       map[string]Llm
	defaultProvider Llm

	httpClient *http.Client

	defaultModel string
	apiKey       string
	saveContext  bool
}

// New creates a new LlmAdapter with the given options.
// It initializes the specified LLM provider and returns a configured adapter.
//
// Example usage:
//
//	adapter, err := llmadapter.New(
//		llmadapter.WithOpenAI("your-api-key"),
//		llmadapter.WithDefaultModel("gpt-4"),
//	)
func New(opts ...llmOption) (*LlmAdapter, error) {
	llm := LlmAdapter{
		providers: make(map[string]Llm),
	}

	for _, opt := range opts {
		opt(&llm)
	}

	for name, provider := range llm.providers {
		if err := provider.Init(llm); err != nil {
			return nil, errors.Wrapf(err, "could not initialize LLM provider '%s'", name)
		}
	}

	return &llm, nil
}

// ResetContext clears the conversation history maintained by the adapter.
// This is useful when you want to start a new conversation without creating a
// new adapter instance. This also clears the systems instructions.
func (llm *LlmAdapter) ResetContext(providers ...string) {
	if len(providers) == 0 {
		llm.defaultProvider.ResetContext()
	}

	for _, provider := range providers {
		if _, ok := llm.providers[provider]; ok {
			llm.providers[provider].ResetContext()
		}
	}
}

func (llm *LlmAdapter) GetProvider(requestProvider *string) (Llm, error) {
	provider := llm.defaultProvider

	if requestProvider != nil {
		p, ok := llm.providers[*requestProvider]
		if !ok {
			return nil, errors.Newf("unknown provider '%s'", *requestProvider)
		}

		provider = p
	}

	return provider, nil
}

func (llm LlmAdapter) DefaultModel() string {
	return llm.defaultModel
}

func (llm LlmAdapter) ApiKey() string {
	return llm.apiKey
}

func (llm LlmAdapter) SaveContext() bool {
	return llm.saveContext
}
func (llm LlmAdapter) HttpClient() *http.Client {
	return llm.httpClient
}

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

// Llm defines the interface that all LLM providers must implement.
// It provides a contract for initializing, managing context,
// and performing chat completions with different language models.
type Llm interface {
	SetName(string)
	// Init initializes the LLM provider with the given adapter configuration.
	// It is called once when the provider is added to the adapter.
	Init(llm internal.Adapter) error
	// ResetContext clears the conversation history for the specific LLM provider.
	// This allows starting a new conversation without re-initializing the provider.
	ResetThread(*ThreadId)
	// CopyThread copies all history from the provided thread into a new, discrete one.
	CopyThread(*ThreadId) *ThreadId
	// CloseThread deletes a thread and associated resources.
	CloseThread(*ThreadId)
	// ChatCompletion sends a chat completion request to the LLM provider.
	// It takes a context, the adapter's internal configuration, and a Requester
	// to retrieve the request.
	ChatCompletion(context.Context, internal.Adapter, Requester) (*InnerResponse, error)
	// RequestOptionsType returns the reflect.Type of the provider-specific
	// request options struct. This is used for type checking and reflection
	// when processing custom request options.
	RequestOptionsType() reflect.Type

	SubmitBatch(context.Context, internal.Adapter, ...Requester) (*UntypedBatchPromise, error)
	Check(context.Context, *UntypedBatchPromise) (BatchStatus, error)
	Wait(ctx context.Context, pr *UntypedBatchPromise) <-chan BatchWaitResponse
}

// LlmAdapter is the main entrypoint for interacting with different LLM providers.
// It provides a unified interface to send requests and receive responses.
type LlmAdapter struct {
	providers       map[string]Llm
	defaultProvider Llm

	httpClient   *http.Client
	defaultModel string
}

// New creates a new LlmAdapter with the given options.
// It initializes the specified LLM provider and returns a configured adapter.
//
// Example usage:
//
//	adapter, err := llmadapter.New(
//		llmadapter.WithDefaultProvider(provider),
//		llmadapter.WithDefaultModel("gpt-4"),
//		llmadapter.WithApiKey("...")
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
// If called without arguments, will clear the history of the default provider,
// otherwise, it accepts variadic provider names for which to clear the history.
// func (llm *LlmAdapter) ResetThreads(threadIds ...*ThreadId) {
// 	for _, thread := range threadIds {
// 		thread.provider.ResetThread(thread)
// 	}
// }

// GetProvider retrieves an LLM provider based on the given provider name.
// It accepts the provider requested in a specific request, which will override
// the default provider. If the provider argument is nil, it will return the
// configured default provider.
func (llm *LlmAdapter) GetProvider(requestProvider *string) (Llm, error) {
	if llm.defaultProvider == nil {
		return nil, errors.New("no provider was configured")
	}

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

func (llm *LlmAdapter) SubmitBatch(ctx context.Context, providerName string, reqs ...Requester) (*UntypedBatchPromise, error) {
	p, ok := llm.providers[providerName]
	if !ok {
		return nil, errors.Newf("unknown provider '%s'", providerName)
	}

	return p.SubmitBatch(ctx, llm, reqs...)
}

func (llm *LlmAdapter) BatchPromise(providerName string, id string) (*UntypedBatchPromise, error) {
	provider, ok := llm.providers[providerName]
	if !ok {
		return nil, errors.New("cannot find the provider that created this promise")
	}

	return &UntypedBatchPromise{
		ProviderName: providerName,
		Provider:     provider,
		Id:           id,
	}, nil
}

// LlmAdapter implementation of Adapter

func (llm LlmAdapter) DefaultModel() string {
	return llm.defaultModel
}

func (llm LlmAdapter) HttpClient() *http.Client {
	return llm.httpClient
}

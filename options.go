package llmadapter

import "net/http"

type llmOption func(*LlmAdapter)

// WithDefaultProvider sets what LLM provider to use for communication.
func WithDefaultProvider(provider Llm) llmOption {
	return func(llm *LlmAdapter) {
		llm.providers[defaultProvider] = provider
		llm.defaultProvider = llm.providers[defaultProvider]
	}
}

// WithProvider registers a provider.
//
// The first one to be registered will become the default, unless a default was
// already or is defined later with `SetDefaultProvider`.
func WithProvider(name string, provider Llm) llmOption {
	return func(llm *LlmAdapter) {
		llm.providers[name] = provider

		if llm.defaultProvider == nil {
			llm.defaultProvider = llm.providers[name]
			llm.providers[defaultProvider] = llm.providers[name]
		}
	}
}

// WithDefaultModel sets the model to use if not specified in a particular
// request. It is the caller's responsibility to ensure the requested model is
// available on the configured provider.
func WithDefaultModel(model string) llmOption {
	return func(llm *LlmAdapter) {
		llm.defaultModel = model
	}
}

// WithApiKey sets an API key to use for every request to the provider. Note
// that not all providers support API key authentication and a given provider
// will only use this if it requires it or ignore it otherwise.
//
// For provider needing specific authentication, an specific option will be
// available on the provider itself.
func WithApiKey(key string) llmOption {
	return func(llm *LlmAdapter) {
		llm.apiKey = key
	}
}

// WithSaveContext enables history accumulation.
//
// When enabled, any messages sent to and received from the provider will be
// recorded to be reused in subsequent requests. If this is not called, every
// request to the provider will have a blank context.
func WithSaveContext() llmOption {
	return func(llm *LlmAdapter) {
		llm.saveContext = true
	}
}

// WithHttpClient sets a custom HTTP clients to be used.
//
// If a provider does not support overriding the HTTP client, this will be
// ignored.
func WithHttpClient(client *http.Client) llmOption {
	return func(llm *LlmAdapter) {
		llm.httpClient = client
	}
}

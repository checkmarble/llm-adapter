package llmberjack

import "net/http"

type llmOption func(*llmberjack)

// WithDefaultProvider sets what LLM provider to use for communication.
func WithDefaultProvider(provider Llm) llmOption {
	return func(llm *llmberjack) {
		llm.providers[defaultProvider] = provider
		llm.defaultProvider = llm.providers[defaultProvider]
	}
}

// WithProvider registers a provider.
//
// The first one to be registered will become the default, unless a default was
// already or is defined later with `SetDefaultProvider`.
func WithProvider(name string, provider Llm) llmOption {
	return func(llm *llmberjack) {
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
	return func(llm *llmberjack) {
		llm.defaultModel = model
	}
}

// WithHttpClient sets a custom HTTP clients to be used.
//
// If a provider does not support overriding the HTTP client, this will be
// ignored.
func WithHttpClient(client *http.Client) llmOption {
	return func(llm *llmberjack) {
		llm.httpClient = client
	}
}

package internal

import "net/http"

// Adapter defines the interface for internal configuration and utility methods
// that LLM providers can access from the main llmberjack.
type Adapter interface {
	// DefaultModel returns the default model name configured for the adapter.
	DefaultModel() string
	// HttpClient returns the *http.Client instance used for making HTTP requests.
	HttpClient() *http.Client
}

// ProviderRequestOptions is a marker interface that all provider-specific
// request options structs must implement. This allows for type assertion
// and reflection to extract provider-specific options from a generic request.
type ProviderRequestOptions interface {
	// ProviderOptions is a dummy method used to satisfy the interface.
	// It has no functional purpose other than to mark a struct as containing
	// provider-specific request options.
	ProviderRequestOptions()
}

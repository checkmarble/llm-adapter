package llmadapter

import (
	"github.com/cockroachdb/errors"
)

// LlmAdapter is the main entrypoint for interacting with different LLM providers.
// It provides a unified interface to send requests and receive responses.
type LlmAdapter struct {
	provider Llm

	defaultModel string
	apiKey       string
	saveContext  bool
}

// NewLlmAdapter creates a new LlmAdapter with the given options.
// It initializes the specified LLM provider and returns a configured adapter.
//
// Example usage:
//
//	adapter, err := llmadapter.NewLlmAdapter(
//		llmadapter.WithOpenAI("your-api-key"),
//		llmadapter.WithDefaultModel("gpt-4"),
//	)
func NewLlmAdapter(opts ...llmOption) (*LlmAdapter, error) {
	llm := LlmAdapter{}

	for _, opt := range opts {
		opt(&llm)
	}

	if err := llm.provider.Init(llm); err != nil {
		return nil, errors.Wrap(err, "could not initialize LLM provider")
	}

	return &llm, nil
}

// ResetContext clears the conversation history maintained by the adapter.
// This is useful when you want to start a new conversation without creating a
// new adapter instance. This also clears the systems instructions.
func (llm *LlmAdapter) ResetContext() {
	llm.provider.ResetContext()
}

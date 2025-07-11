package llmadapter

import (
	"context"

	"github.com/cockroachdb/errors"
)

type LlmAdapter struct {
	provider Llm

	DefaultModel string
	ApiKey       string
	SaveContext  bool
}

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

func (llm *LlmAdapter) ChatCompletion(ctx context.Context, r InnerRequest) (*Response, error) {
	return llm.provider.ChatCompletions(ctx, llm, r)
}

func (llm *LlmAdapter) ResetContext() {
	llm.provider.ResetContext()
}

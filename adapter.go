package llmadapter

import (
	"context"
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
		return nil, err
	}

	return &llm, nil
}

func (llm *LlmAdapter) ChatCompletion(ctx context.Context, r Request) (*Response, error) {
	return llm.provider.ChatCompletions(ctx, llm, r)
}

func (llm *LlmAdapter) ResetContext() {
	llm.provider.ResetContext()
}

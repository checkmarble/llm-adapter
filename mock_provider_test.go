package llmadapter

import (
	"context"
)

type MockProvider struct{}

func NewMockProvider() (*MockProvider, error) {
	return &MockProvider{}, nil
}

func (p *MockProvider) Init(llm Adapter) error {
	return nil
}

func (p *MockProvider) ResetContext() {
}

func (MockProvider) ChatCompletion(ctx context.Context, llm Adapter, requester LlmRequester) (*Response, error) {
	return nil, nil
}

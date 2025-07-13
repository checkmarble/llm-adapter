package llmadapter

import (
	"context"
	"reflect"

	"github.com/checkmarble/marble-llm-adapter/internal"
)

type MockProvider struct{}

func (MockProvider) RequestOptionsType() reflect.Type {
	return nil
}

func NewMockProvider() (*MockProvider, error) {
	return &MockProvider{}, nil
}

func (p *MockProvider) Init(llm internal.Adapter) error {
	return nil
}

func (p *MockProvider) ResetContext() {
}

func (MockProvider) ChatCompletion(ctx context.Context, llm internal.Adapter, requester LlmRequester) (*Response, error) {
	return nil, nil
}

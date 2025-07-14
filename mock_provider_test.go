package llmadapter

import (
	"context"
	"reflect"

	"github.com/checkmarble/marble-llm-adapter/internal"
)

type MockMessage struct {
	Text string
}

type MockProvider struct {
	History History[MockMessage]
}

func (MockProvider) RequestOptionsType() reflect.Type {
	return nil
}

func NewMockProvider() (*MockProvider, error) {
	return &MockProvider{
		History: History[MockMessage]{
			history: make([]MockMessage, 0),
		},
	}, nil
}

func (p *MockProvider) Init(llm internal.Adapter) error {
	return nil
}

func (p *MockProvider) ResetContext() {
	p.History.Clear()
}

func (p *MockProvider) ChatCompletion(ctx context.Context, llm internal.Adapter, requester Requester) (*InnerResponse, error) {
	msg := MockMessage{"Hello, world!"}

	return &InnerResponse{
		Candidates: []ResponseCandidate{
			{
				SelectCandidate: func() {
					p.History.Save(msg)
				},
			},
		},
	}, nil
}

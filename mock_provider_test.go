package llmadapter

import (
	"context"
	"reflect"

	"github.com/checkmarble/marble-llm-adapter/internal"
	"github.com/stretchr/testify/mock"
)

type MockMessage struct {
	Text string
}

type MockProvider struct {
	mock.Mock

	History History[MockMessage]
}

func (*MockProvider) RequestOptionsType() reflect.Type {
	return nil
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		History: History[MockMessage]{
			history: make([]MockMessage, 0),
		},
	}
}

func (p *MockProvider) Init(llm internal.Adapter) error {
	return p.Called(llm).Error(0)
}

func (p *MockProvider) ResetContext() {
	p.History.Clear()
}

func (p *MockProvider) ChatCompletion(ctx context.Context, llm internal.Adapter, requester Requester) (*InnerResponse, error) {
	args := p.Called(ctx, llm, requester)

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	msg := args.Get(0).(MockMessage)

	return &InnerResponse{
		Candidates: []ResponseCandidate{
			{
				Text: msg.Text,
				SelectCandidate: func() {
					p.History.Save(msg)
				},
			},
		},
	}, nil
}

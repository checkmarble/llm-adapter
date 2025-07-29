package llmadapter

import (
	"context"
	"io"
	"reflect"

	"github.com/checkmarble/llm-adapter/internal"
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
			history: make(map[*ThreadId][]MockMessage, 0),
		},
	}
}

func (p *MockProvider) Init(llm internal.Adapter) error {
	return p.Called(llm).Error(0)
}

func (p *MockProvider) ResetThread(threadId *ThreadId) {
	p.History.Clear(threadId)
}

func (p *MockProvider) CopyThread(threadId *ThreadId) *ThreadId {
	return p.History.Copy(threadId)
}

func (p *MockProvider) CloseThread(threadId *ThreadId) {
	p.History.Close(threadId)
}

func (p *MockProvider) ChatCompletion(ctx context.Context, llm internal.Adapter, requester Requester) (*InnerResponse, error) {
	args := p.Called(ctx, llm, requester)

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	req := requester.ToRequest()

	for _, msg := range req.Messages {
		for _, part := range msg.Parts {
			f, err := io.ReadAll(part)
			if err != nil {
				return nil, err
			}

			if req.ThreadId != nil {
				p.History.Save(req.ThreadId, MockMessage{string(f)})
			}
		}
	}

	msg := args.Get(0).(MockMessage)

	return &InnerResponse{
		Candidates: []ResponseCandidate{
			{
				Text: msg.Text,
				SelectCandidate: func() {
					if req.ThreadId != nil {
						p.History.Save(req.ThreadId, msg)
					}
				},
			},
		},
	}, nil
}

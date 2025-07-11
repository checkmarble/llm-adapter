package llmadapter

import (
	"io"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestToolCalled(t *testing.T) {
	type Args struct {
		Integer int `json:"integer"`
	}

	called := 0

	tool := NewTool[Args]("name", "", Function(func(args Args) (string, error) {
		called += args.Integer

		return "called", nil
	}))

	resp := TypedResponse[struct{}]{
		Response: Response{
			Candidates: []ResponseCandidate{{
				ToolCalls: []ResponseToolCall{
					{
						Id:         "id",
						Name:       "name",
						Parameters: []byte(`{"integer": 10}`),
					},
				},
				SelectCandidate: func() {},
			}},
		},
	}

	req := NewUntypedRequest().FromCandidate(resp, 0).WithToolExecution(tool)

	assert.Nil(t, req.err)
	assert.Equal(t, 10, called)
	assert.Len(t, req.Messages, 1)
	assert.NotNil(t, req.Messages[0].Tool)
	assert.Equal(t, "id", req.Messages[0].Tool.Id)
	assert.Equal(t, "name", req.Messages[0].Tool.Name)
	assert.Equal(t, RoleTool, req.Messages[0].Role)
	assert.Equal(t, TypeText, req.Messages[0].Type)
	assert.Len(t, req.Messages[0].Parts, 1)

	content, err := io.ReadAll(req.Messages[0].Parts[0])

	assert.Nil(t, err)
	assert.Equal(t, "called", string(content))
}

func TestToolNotCalled(t *testing.T) {
	type Args struct {
		Integer int `json:"integer"`
	}

	called := 0

	tool := NewTool[Args]("name", "", Function(func(args Args) (string, error) {
		called += args.Integer

		return "called", nil
	}))

	resp := TypedResponse[struct{}]{
		Response: Response{
			Candidates: []ResponseCandidate{{
				ToolCalls: []ResponseToolCall{
					{
						Id:         "id",
						Name:       "invalidname",
						Parameters: []byte(`{"integer": 10}`),
					},
				},
				SelectCandidate: func() {},
			}},
		},
	}

	req := NewUntypedRequest().FromCandidate(resp, 0).WithToolExecution(tool)

	assert.NotNil(t, req.err)
	assert.Contains(t, req.err.Error(), "no tool was registered")
	assert.Equal(t, 0, called)
}

func TestToolError(t *testing.T) {
	type Args struct {
		Integer int `json:"integer"`
	}

	called := 0

	tool := NewTool[Args]("name", "", Function(func(args Args) (string, error) {
		called += args.Integer

		return "called", errors.New("something went wrong")
	}))

	resp := TypedResponse[struct{}]{
		Response: Response{
			Candidates: []ResponseCandidate{{
				ToolCalls: []ResponseToolCall{
					{
						Id:         "id",
						Name:       "name",
						Parameters: []byte(`{"integer": 10}`),
					},
				},
				SelectCandidate: func() {},
			}},
		},
	}

	req := NewUntypedRequest().FromCandidate(resp, 0).WithToolExecution(tool)

	assert.NotNil(t, req.err)
	assert.Contains(t, req.err.Error(), "something went wrong")
	assert.Equal(t, 10, called)
}

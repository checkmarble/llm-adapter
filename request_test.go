package llmberjack

import (
	"io"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func assertParts(t *testing.T, parts []io.Reader, expected string) {
	var sb strings.Builder

	for _, part := range parts {
		s, err := io.ReadAll(part)

		assert.Nil(t, err)

		sb.Write(s)
	}

	assert.Equal(t, expected, sb.String())
}

func TestNewRequest(t *testing.T) {
	type Type struct {
		Number int
	}

	req := NewRequest[Type]()

	assert.Nil(t, req.err)
	assert.Nil(t, req.Model)
	assert.Len(t, req.Messages, 0)
	assert.Nil(t, req.MaxCandidates)
	assert.Nil(t, req.MaxTokens)
	assert.Nil(t, req.Temperature)
	assert.Nil(t, req.TopP)

	req = req.
		WithModel("themodel").
		WithMaxCandidates(1).
		WithMaxTokens(2).
		WithTemperature(3.0).
		WithTopP(4.0).
		WithInstruction("system prompt", "second system prompt").
		WithInstructionReader(strings.NewReader("reader system prompt")).
		WithText(RoleUser, "user prompt", "second user prompt").
		WithTextReader(RoleUser, strings.NewReader("reader user prompt"))

	assert.Nil(t, req.err)
	assert.Equal(t, "themodel", *req.Model)
	assert.Len(t, req.Messages, 4)
	assert.Equal(t, 1, *req.MaxCandidates)
	assert.Equal(t, 2, *req.MaxTokens)
	assert.Equal(t, 3.0, *req.Temperature)
	assert.Equal(t, 4.0, *req.TopP)

	assertParts(t, req.Messages[0].Parts, "system promptsecond system prompt")
	assertParts(t, req.Messages[1].Parts, "reader system prompt")
	assertParts(t, req.Messages[2].Parts, "user promptsecond user prompt")
	assertParts(t, req.Messages[3].Parts, "reader user prompt")
}

func TestRequestWithError(t *testing.T) {
	req := Request[string]{
		err: errors.New("request error"),
	}

	llm, _ := New()

	out, err := req.Do(t.Context(), llm)

	assert.Nil(t, out)
	assert.ErrorContains(t, err, "request error")
}

func TestErrorFromLlm(t *testing.T) {
	p := NewMockProvider()
	p.On("Init", mock.Anything).Return(nil)
	p.On("ChatCompletion", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("provider error"))

	llm, _ := New(WithDefaultProvider(p))
	resp, err := NewUntypedRequest().Do(t.Context(), llm)

	assert.Nil(t, resp)
	assert.ErrorContains(t, err, "provider error")
}

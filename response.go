package llmadapter

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
)

// Candidater represents a type that can have several candidates.
type Candidater interface {
	NumCandidates() int
	Candidate(int) (*ResponseCandidate, error)
}

// Response is a response from an LLM provider.
type Response struct {
	Model      string
	Candidates []ResponseCandidate
}

// ResponseCandidate represent a response from an LLM provider.
type ResponseCandidate struct {
	Text            string
	ToolCalls       []ResponseToolCall
	SelectCandidate func()
}

// ResponseToolCall is a request from an LLM provider to execute a tool.
type ResponseToolCall struct {
	Id         string
	Name       string
	Parameters []byte
}

type TypedResponse[T any] struct {
	Response
}

func (r TypedResponse[T]) NumCandidates() int {
	return len(r.Candidates)
}

func (r TypedResponse[T]) Candidate(idx int) (*ResponseCandidate, error) {
	if idx > len(r.Candidates)-1 {
		return nil, errors.Newf("candidate %d does not exist (%d candidates)", idx, len(r.Candidates))
	}

	return &r.Candidates[idx], nil
}

func (r TypedResponse[T]) Get(idx int) (T, error) {
	if idx > len(r.Candidates)-1 {
		return *new(T), errors.Newf("candidate %d does not exist (%d candidates)", idx, len(r.Candidates))
	}

	candidate := r.Candidates[idx]

	switch any(*new(T)).(type) {
	case string:
		return any(candidate.Text).(T), nil

	default:
		output := new(T)

		if err := json.Unmarshal([]byte(candidate.Text), output); err != nil {
			return *new(T), errors.Wrap(err, "failed to decode response to schema")
		}

		return *output, nil
	}
}

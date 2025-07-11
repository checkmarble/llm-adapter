package llmadapter

import (
	"encoding/json"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

type Candidater interface {
	NumCandidates() int
	Candidate(int) (*ResponseCandidate, error)
}

type Response struct {
	Model      string
	Candidates []ResponseCandidate
}

type ResponseCandidate struct {
	Text                []string
	ToolCalls           []ResponseToolCall
	SelectCandidateFunc func()
}

type ResponseToolCall struct {
	Id         string
	Name       string
	Parameters []byte
}

func (r Response) SelectCandidate(idx int) []string {
	r.Candidates[idx].SelectCandidateFunc()

	return r.Candidates[idx].Text
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

func (r TypedResponse[T]) Get(idx int) ([]T, error) {
	if idx > len(r.Candidates)-1 {
		return nil, errors.Newf("candidate %d does not exist (%d candidates)", idx, len(r.Candidates))
	}

	candidate := r.Candidates[idx]

	switch any(*new(T)).(type) {
	case string:
		return any(lo.Map(candidate.Text, func(s string, _ int) string {
			return s
		})).([]T), nil

	default:
		outputs := make([]T, len(candidate.Text))

		for idx, item := range candidate.Text {
			if err := json.Unmarshal([]byte(item), &outputs[idx]); err != nil {
				return nil, errors.Wrap(err, "failed to decode response to schema")
			}
		}

		return outputs, nil
	}
}

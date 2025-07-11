package llmadapter

import (
	"encoding/json"

	"github.com/samber/lo"
)

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

func (r TypedResponse[T]) Get(idx int) ([]T, error) {
	switch any(*new(T)).(type) {
	case string:
		return any(lo.Map(r.Candidates[idx].Text, func(s string, _ int) string {
			return s
		})).([]T), nil

	default:
		outputs := make([]T, len(r.Candidates[idx].Text))

		for idx, item := range r.Candidates[idx].Text {
			if err := json.Unmarshal([]byte(item), &outputs[idx]); err != nil {
				return nil, err
			}
		}

		return outputs, nil
	}
}

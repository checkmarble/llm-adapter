package llmadapter_test

import (
	"testing"

	llmadapter "github.com/checkmarble/llm-adapter"
	"github.com/stretchr/testify/assert"
)

func TestResponseGetOutput(t *testing.T) {
	type output struct {
		Text string `json:"text"`
	}

	r := llmadapter.Response[output]{
		InnerResponse: llmadapter.InnerResponse{
			Candidates: []llmadapter.ResponseCandidate{
				{Text: `{"text":"first response"}`},
				{Text: `{"text":"second response"}`},
			},
		},
	}

	assert.Equal(t, r.NumCandidates(), 2)

	c, err := r.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "first response", c.Text)

	c, err = r.Get(1)

	assert.Nil(t, err)
	assert.Equal(t, "second response", c.Text)

	c, err = r.Get(2)

	assert.ErrorContains(t, err, "candidate 2 does not exist")
}

func TestResponseGetInvalidJson(t *testing.T) {
	type output struct {
		Text string `json:"text"`
	}

	r := llmadapter.Response[output]{
		InnerResponse: llmadapter.InnerResponse{
			Candidates: []llmadapter.ResponseCandidate{
				{Text: `{"text":"first`},
			},
		},
	}

	assert.Equal(t, r.NumCandidates(), 1)

	_, err := r.Get(0)

	assert.ErrorContains(t, err, "failed to decode response to schema")
}

func TestResponseGetStringOutput(t *testing.T) {
	r := llmadapter.Response[string]{
		InnerResponse: llmadapter.InnerResponse{
			Candidates: []llmadapter.ResponseCandidate{
				{Text: "first response"},
				{Text: "second response"},
			},
		},
	}

	assert.Equal(t, r.NumCandidates(), 2)

	c, err := r.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "first response", c)

	c, err = r.Get(1)

	assert.Nil(t, err)
	assert.Equal(t, "second response", c)

	_, err = r.Get(2)

	assert.ErrorContains(t, err, "candidate 2 does not exist")
}

func TestResponseGetCandidate(t *testing.T) {
	type output struct {
		Text string `json:"text"`
	}

	r := llmadapter.Response[output]{
		InnerResponse: llmadapter.InnerResponse{
			Candidates: []llmadapter.ResponseCandidate{
				{Text: `{"text":"first response"}`},
				{Text: `{"text":"second response"}`},
			},
		},
	}

	assert.Equal(t, r.NumCandidates(), 2)

	c, err := r.Candidate(0)

	assert.Nil(t, err)
	assert.Equal(t, r.Candidates[0], *c)

	_, err = r.Candidate(2)

	assert.ErrorContains(t, err, "candidate 2 does not exist")
}

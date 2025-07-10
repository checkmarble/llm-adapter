package aistudio

import (
	"context"
	"io"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/samber/lo"
	"google.golang.org/genai"
)

type AiStudio struct {
	client  *genai.Client
	history llmadapter.History[*genai.Content]
}

func New(opts ...llmOption) (*AiStudio, error) {
	llm := AiStudio{}

	for _, opt := range opts {
		opt(&llm)
	}

	return &llm, nil
}

func (llm *AiStudio) Init(adapter llmadapter.LlmAdapter) error {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  adapter.ApiKey,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		return err
	}

	llm.client = client

	return nil
}

func (p *AiStudio) ResetContext() {
	p.history.Clear()
}

func (p *AiStudio) ChatCompletions(ctx context.Context, llm *llmadapter.LlmAdapter, r llmadapter.Request) (*llmadapter.Response, error) {
	contents := make([]*genai.Content, 0, len(r.Messages))

	if llm.SaveContext {
		contents = append(contents, p.history.Load()...)
	}

	model := llm.DefaultModel
	if r.Model != nil {
		model = *r.Model
	}

	cfg := genai.GenerateContentConfig{}

Messages:
	for _, msg := range r.Messages {
		parts := make([]*genai.Part, 0, len(msg.Parts))

		for _, part := range msg.Parts {
			buf, _ := io.ReadAll(part)

			switch msg.Type {
			case llmadapter.TypeText:
				parts = append(parts, genai.NewPartFromText(string(buf)))
			}
		}

		role := genai.RoleUser

		switch msg.Role {
		case llmadapter.RoleAi:
			role = genai.RoleModel
		case llmadapter.RoleUser:
			role = genai.RoleUser
		case llmadapter.RoleSystem:
			cfg.SystemInstruction = &genai.Content{
				Parts: parts,
			}

			if llm.SaveContext {
				p.history.Save(cfg.SystemInstruction)
			}

			continue Messages
		}

		content := &genai.Content{
			Role:  role,
			Parts: parts,
		}

		if llm.SaveContext {
			p.history.Save(content)
		}

		contents = append(contents, content)
	}

	response, err := p.client.Models.GenerateContent(ctx, model, contents, &cfg)

	if err != nil {
		return nil, err
	}

	resp := llmadapter.Response{
		Model:      response.ModelVersion,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Candidates)),
	}

	for idx, candidate := range response.Candidates {
		if llm.SaveContext {
			p.history.Save(candidate.Content)
		}

		resp.Candidates[idx] = llmadapter.ResponseCandidate{
			Text: lo.Map(candidate.Content.Parts, func(part *genai.Part, index int) string {
				return part.Text
			}),
		}
	}

	return &resp, nil
}

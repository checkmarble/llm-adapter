package aistudio

import (
	"context"
	"encoding/json"
	"io"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"google.golang.org/genai"
)

type AiStudio struct {
	client  *genai.Client
	history llmadapter.History[*genai.Content]

	backend  genai.Backend
	project  string
	location string
}

func New(opts ...opt) (*AiStudio, error) {
	llm := AiStudio{
		backend: genai.BackendGeminiAPI,
	}

	for _, opt := range opts {
		opt(&llm)
	}

	return &llm, nil
}

func (p *AiStudio) Init(adapter llmadapter.LlmAdapter) error {
	cfg := genai.ClientConfig{
		Project:  p.project,
		Location: p.location,
	}

	if p.backend != genai.BackendUnspecified {
		cfg.Backend = p.backend
	}
	if cfg.Backend == genai.BackendGeminiAPI {
		cfg.APIKey = adapter.ApiKey
	}

	client, err := genai.NewClient(context.Background(), &cfg)
	if err != nil {
		return err
	}

	p.client = client

	return nil
}

func (p *AiStudio) ResetContext() {
	p.history.Clear()
}

func (p *AiStudio) ChatCompletions(ctx context.Context, llm *llmadapter.LlmAdapter, r llmadapter.InnerRequest) (*llmadapter.Response, error) {
	contents := make([]*genai.Content, 0, len(r.Messages))

	if llm.SaveContext {
		contents = append(contents, p.history.Load()...)
	}

	model := llm.DefaultModel
	if r.Model != nil {
		model = *r.Model
	}

	cfg := genai.GenerateContentConfig{}

	if r.ResponseSchema != nil {
		cfg.ResponseMIMEType = "application/json"
		cfg.ResponseJsonSchema = r.ResponseSchema.Schema
	}

	cfg.Tools = lo.MapToSlice(r.Tools, func(fn string, t llmadapter.Tool) *genai.Tool {
		return &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:                 t.Name,
					Description:          t.Description,
					ParametersJsonSchema: t.Parameters,
				},
			},
		}
	})

Messages:
	for _, msg := range r.Messages {
		parts := make([]*genai.Part, 0, len(msg.Parts))

		for _, part := range msg.Parts {
			buf, err := io.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err, "could not read content part")
			}

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
		case llmadapter.RoleTool:
			msg := &genai.Content{
				Role: role,
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       msg.Tool.Id,
							Name:     msg.Tool.Name,
							Response: map[string]any{"output": parts[0]},
						},
					},
				},
			}

			contents = append(contents, msg)

			if llm.SaveContext {
				p.history.Save(msg)
			}

			continue Messages
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
		return nil, errors.Wrap(err, "LLM provider failed to generate content")
	}

	resp := llmadapter.Response{
		Model:      response.ModelVersion,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Candidates)),
	}

	for idx, candidate := range response.Candidates {
		toolCalls := make([]llmadapter.ResponseToolCall, len(response.FunctionCalls()))

		for idx, toolCall := range response.FunctionCalls() {
			params, err := json.Marshal(toolCall.Args)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse tool call parameters")
			}

			toolCalls[idx] = llmadapter.ResponseToolCall{
				Id:         toolCall.ID,
				Name:       toolCall.Name,
				Parameters: params,
			}
		}

		resp.Candidates[idx] = llmadapter.ResponseCandidate{
			Text: lo.Map(candidate.Content.Parts, func(part *genai.Part, index int) string {
				return part.Text
			}),
			ToolCalls: toolCalls,
			SelectCandidateFunc: func() {
				if llm.SaveContext {
					p.history.Save(candidate.Content)
				}
			},
		}
	}

	return &resp, nil
}

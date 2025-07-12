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
	model    *string
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

func (p *AiStudio) Init(adapter llmadapter.Adapter) error {
	cfg := genai.ClientConfig{
		Backend: genai.BackendGeminiAPI,
	}

	if p.backend != genai.BackendUnspecified {
		cfg.Backend = p.backend
	}
	switch cfg.Backend {
	case genai.BackendGeminiAPI:
		cfg.APIKey = adapter.ApiKey()
	case genai.BackendVertexAI:
		cfg.Project = p.project
		cfg.Location = p.location
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

func (p *AiStudio) ChatCompletion(ctx context.Context, llm llmadapter.Adapter, requester llmadapter.LlmRequester) (*llmadapter.Response, error) {
	model, ok := lo.Coalesce(requester.ToRequest().Model, p.model, lo.ToPtr(llm.DefaultModel()))
	if !ok {
		return nil, errors.New("no model was configured")
	}

	contents, cfg, err := p.adaptRequest(llm, requester)
	if err != nil {
		return nil, errors.Wrap(err, "could not adapt request")
	}

	response, err := p.client.Models.GenerateContent(ctx, *model, contents, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "LLM provider failed to generate content")
	}

	return p.adaptResponse(llm, response)
}

func (p *AiStudio) adaptRequest(llm llmadapter.Adapter, requester llmadapter.LlmRequester) ([]*genai.Content, *genai.GenerateContentConfig, error) {
	r := requester.ToRequest()
	contents := make([]*genai.Content, 0, len(r.Messages))

	if llm.SaveContext() {
		contents = append(contents, p.history.Load()...)
	}

	cfg := genai.GenerateContentConfig{}

	if r.Grounding {
		cfg.Tools = []*genai.Tool{{
			GoogleSearch: &genai.GoogleSearch{},
		}}
	}

	if r.ResponseSchema != nil {
		cfg.ResponseMIMEType = "application/json"
		cfg.ResponseJsonSchema = r.ResponseSchema.Schema
	}

	cfg.Tools = append(cfg.Tools, lo.MapToSlice(r.Tools, func(_ string, t llmadapter.Tool) *genai.Tool {
		return &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:                 t.Name,
					Description:          t.Description,
					ParametersJsonSchema: t.Parameters,
				},
			},
		}
	})...)

Messages:
	for _, msg := range r.Messages {
		parts := make([]*genai.Part, 0, len(msg.Parts))

		for _, part := range msg.Parts {
			buf, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, errors.Wrap(err, "could not read content part")
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

			if llm.SaveContext() {
				p.history.Save(msg)
			}

			continue Messages
		case llmadapter.RoleSystem:
			if cfg.SystemInstruction == nil {
				cfg.SystemInstruction = &genai.Content{
					Parts: make([]*genai.Part, 0),
				}
			}

			cfg.SystemInstruction.Parts = append(cfg.SystemInstruction.Parts, parts...)

			if llm.SaveContext() {
				p.history.Save(cfg.SystemInstruction)
			}

			continue Messages
		}

		content := &genai.Content{
			Role:  role,
			Parts: parts,
		}

		if llm.SaveContext() {
			p.history.Save(content)
		}

		contents = append(contents, content)
	}

	return contents, &cfg, nil
}

func (p *AiStudio) adaptResponse(llm llmadapter.Adapter, response *genai.GenerateContentResponse) (*llmadapter.Response, error) {
	resp := llmadapter.Response{
		Model:      response.ModelVersion,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Candidates)),
	}

	for idx, candidate := range response.Candidates {
		if len(candidate.Content.Parts) == 0 {
			return nil, errors.New("LLM provider generated no content")
		}

		var grounding *llmadapter.ResponseGrounding

		if candidate.GroundingMetadata != nil {
			grounding = &llmadapter.ResponseGrounding{
				Searches: candidate.GroundingMetadata.WebSearchQueries,
				Sources: lo.Map(candidate.GroundingMetadata.GroundingChunks, func(c *genai.GroundingChunk, _ int) llmadapter.ResponseGroudingSource {
					return llmadapter.ResponseGroudingSource{
						Domain: c.Web.Domain,
						Url:    c.Web.URI,
					}
				}),
				Snippets: lo.Map(candidate.GroundingMetadata.GroundingSupports, func(s *genai.GroundingSupport, _ int) string {
					return s.Segment.Text
				}),
			}
		}

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
			Text:      candidate.Content.Parts[0].Text,
			ToolCalls: toolCalls,
			Grounding: grounding,
			SelectCandidate: func() {
				if llm.SaveContext() {
					p.history.Save(candidate.Content)
				}
			},
		}
	}

	return &resp, nil
}

package openai

import (
	"context"
	"io"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/samber/lo"
)

type OpenAi struct {
	client  openai.Client
	history llmadapter.History[openai.ChatCompletionMessageParamUnion]

	baseUrl string
}

func New(opts ...llmOption) (*OpenAi, error) {
	llm := OpenAi{}

	for _, opt := range opts {
		opt(&llm)
	}

	return &llm, nil
}

func (p *OpenAi) Init(adapter llmadapter.LlmAdapter) error {
	opts := []option.RequestOption{
		option.WithAPIKey(adapter.ApiKey),
	}

	if p.baseUrl != "" {
		opts = append(opts, option.WithBaseURL(p.baseUrl))
	}

	p.client = openai.NewClient(opts...)

	return nil
}

func (p *OpenAi) ResetContext() {
	p.history.Clear()
}

func (p *OpenAi) ChatCompletions(ctx context.Context, llm *llmadapter.LlmAdapter, r llmadapter.Request) (*llmadapter.Response, error) {
	contents := make([]openai.ChatCompletionMessageParamUnion, 0, len(r.Messages))

	if llm.SaveContext {
		contents = append(contents, p.history.Load()...)
	}

	model := llm.DefaultModel
	if r.Model != nil {
		model = *r.Model
	}

	cfg := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: contents,
	}

	for _, msg := range r.Messages {
		parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(msg.Parts))

		for _, part := range msg.Parts {
			buf, _ := io.ReadAll(part)

			switch msg.Type {
			case llmadapter.TypeText:
				parts = append(parts, openai.TextContentPart(string(buf)))
			}
		}

		content := openai.ChatCompletionMessageParamUnion{}

		switch msg.Role {
		case llmadapter.RoleAi:
			content.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfArrayOfContentParts: lo.Map(parts, func(p openai.ChatCompletionContentPartUnionParam, _ int) openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion {
						return openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
							OfText: &openai.ChatCompletionContentPartTextParam{
								Text: *p.GetText(),
							},
						}
					}),
				},
			}

		case llmadapter.RoleUser:
			content.OfUser = &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfArrayOfContentParts: lo.Map(parts, func(p openai.ChatCompletionContentPartUnionParam, _ int) openai.ChatCompletionContentPartUnionParam {
						return openai.ChatCompletionContentPartUnionParam{
							OfText: &openai.ChatCompletionContentPartTextParam{
								Text: *p.GetText(),
							},
						}
					}),
				},
			}

		case llmadapter.RoleSystem:
			content.OfSystem = &openai.ChatCompletionSystemMessageParam{
				Content: openai.ChatCompletionSystemMessageParamContentUnion{
					OfArrayOfContentParts: lo.Map(parts, func(p openai.ChatCompletionContentPartUnionParam, _ int) openai.ChatCompletionContentPartTextParam {
						return openai.ChatCompletionContentPartTextParam{
							Text: *p.GetText(),
						}
					}),
				},
			}
		}

		if llm.SaveContext {
			p.history.Save(content)
		}

		cfg.Messages = append(cfg.Messages, content)
	}

	response, err := p.client.Chat.Completions.New(ctx, cfg)

	if err != nil {
		return nil, err
	}

	resp := llmadapter.Response{
		Model:      response.Model,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Choices)),
	}

	for idx, candidate := range response.Choices {
		resp.Candidates[idx] = llmadapter.ResponseCandidate{
			Text: []string{candidate.Message.Content},
			SelectCandidate: func() {
				if llm.SaveContext {
					p.history.Save(openai.AssistantMessage(candidate.Message.Content))
				}
			},
		}
	}

	return &resp, nil
}

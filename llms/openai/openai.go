package openai

import (
	"context"
	"encoding/json"
	"io"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/cockroachdb/errors"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/samber/lo"
)

type OpenAi struct {
	client  openai.Client
	history llmadapter.History[openai.ChatCompletionMessageParamUnion]

	baseUrl string
}

func New(opts ...opt) (*OpenAi, error) {
	llm := OpenAi{}

	for _, opt := range opts {
		opt(&llm)
	}

	return &llm, nil
}

func (p *OpenAi) Init(llm llmadapter.Adapter) error {
	opts := []option.RequestOption{
		option.WithAPIKey(llm.ApiKey()),
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

func (p *OpenAi) ChatCompletion(ctx context.Context, llm llmadapter.Adapter, requester llmadapter.LlmRequester) (*llmadapter.Response, error) {
	r := requester.ToRequest()
	contents := make([]openai.ChatCompletionMessageParamUnion, 0, len(r.Messages))

	if llm.SaveContext() {
		contents = append(contents, p.history.Load()...)
	}

	model := llm.DefaultModel()
	if r.Model != nil {
		model = *r.Model
	}

	cfg := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: contents,
	}

	if r.ResponseSchema != nil {
		cfg.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        r.ResponseSchema.Name,
					Description: openai.String(r.ResponseSchema.Description),
					Schema:      r.ResponseSchema.Schema,
					Strict:      openai.Bool(true),
				},
			},
		}
	}

	for _, tool := range r.Tools {
		paramsJson, err := json.Marshal(tool.Parameters)
		if err != nil {
			return nil, errors.Wrap(err, "failed to encode tool parameters")
		}

		var params map[string]any

		if err := json.Unmarshal(paramsJson, &params); err != nil {
			return nil, errors.Wrap(err, "failed to encode tool parameters")
		}

		cfg.Tools = append(cfg.Tools, openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  openai.FunctionParameters(params),
			},
		})
	}

	for _, msg := range r.Messages {
		parts := make([]openai.ChatCompletionContentPartUnionParam, 0, len(msg.Parts))

		for _, part := range msg.Parts {
			buf, err := io.ReadAll(part)
			if err != nil {
				return nil, errors.Wrap(err, "could not read content part")
			}

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

		case llmadapter.RoleTool:
			content.OfTool = &openai.ChatCompletionToolMessageParam{
				ToolCallID: msg.Tool.Id,
				Content: openai.ChatCompletionToolMessageParamContentUnion{
					OfArrayOfContentParts: lo.Map(parts, func(p openai.ChatCompletionContentPartUnionParam, _ int) openai.ChatCompletionContentPartTextParam {
						return openai.ChatCompletionContentPartTextParam{
							Text: *p.GetText(),
						}
					}),
				},
			}
		}

		if llm.SaveContext() {
			p.history.Save(content)
		}

		cfg.Messages = append(cfg.Messages, content)
	}

	response, err := p.client.Chat.Completions.New(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "LLM provider failed to generate content")
	}

	resp := llmadapter.Response{
		Model:      response.Model,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Choices)),
	}

	for idx, candidate := range response.Choices {
		toolCalls := make([]llmadapter.ResponseToolCall, len(candidate.Message.ToolCalls))

		for idx, toolCall := range candidate.Message.ToolCalls {
			toolCalls[idx] = llmadapter.ResponseToolCall{
				Id:         toolCall.ID,
				Name:       toolCall.Function.Name,
				Parameters: []byte(toolCall.Function.Arguments),
			}
		}

		resp.Candidates[idx] = llmadapter.ResponseCandidate{
			Text:      candidate.Message.Content,
			ToolCalls: toolCalls,
			SelectCandidate: func() {
				if llm.SaveContext() {
					msg := openai.ChatCompletionMessageParamUnion{
						OfAssistant: &openai.ChatCompletionAssistantMessageParam{
							ToolCalls: candidate.Message.ToParam().GetToolCalls(),
							Content: openai.ChatCompletionAssistantMessageParamContentUnion{
								OfString: openai.String(candidate.Message.Content),
							},
						},
					}

					p.history.Save(msg)
				}
			},
		}
	}

	return &resp, nil
}

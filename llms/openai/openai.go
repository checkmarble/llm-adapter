package openai

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"time"

	llmadapter "github.com/checkmarble/llm-adapter"
	"github.com/checkmarble/llm-adapter/internal"
	"github.com/cockroachdb/errors"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/samber/lo"
)

type OpenAi struct {
	client  openai.Client
	history llmadapter.History[openai.ChatCompletionMessageParamUnion]

	RequestHookFunc func(llmadapter.Requester, *openai.ChatCompletionNewParams) error

	baseUrl string
	apiKey  string
	model   *string
}

func (*OpenAi) RequestOptionsType() reflect.Type {
	return nil
}

func New(opts ...Opt) (*OpenAi, error) {
	llm := OpenAi{}

	for _, opt := range opts {
		opt(&llm)
	}

	return &llm, nil
}

func (p *OpenAi) Init(llm internal.Adapter) error {
	opts := []option.RequestOption{
		option.WithAPIKey(p.apiKey),
	}

	if llm.HttpClient() != nil {
		opts = append(opts, option.WithHTTPClient(llm.HttpClient()))
	}
	if p.baseUrl != "" {
		opts = append(opts, option.WithBaseURL(p.baseUrl))
	}

	p.client = openai.NewClient(opts...)

	return nil
}

func (p *OpenAi) ResetThread(threadId *llmadapter.ThreadId) {
	p.history.Clear(threadId)
}

func (p *OpenAi) CopyThread(threadId *llmadapter.ThreadId) *llmadapter.ThreadId {
	return p.history.Copy(threadId)
}

func (p *OpenAi) CloseThread(threadId *llmadapter.ThreadId) {
	p.history.Close(threadId)
}

func (p *OpenAi) ChatCompletion(ctx context.Context, llm internal.Adapter, requester llmadapter.Requester) (*llmadapter.InnerResponse, error) {
	cfg, err := p.adaptRequest(llm, requester)
	if err != nil {
		return nil, errors.Wrap(err, "could not adapt request")
	}

	if p.RequestHookFunc != nil {
		if err := p.RequestHookFunc(requester, cfg); err != nil {
			return nil, err
		}
	}

	response, err := p.client.Chat.Completions.New(ctx, *cfg)
	if err != nil {
		return nil, errors.Wrap(err, "LLM provider failed to generate content")
	}

	return p.adaptResponse(llm, response, requester)
}

func (p *OpenAi) adaptRequest(llm internal.Adapter, requester llmadapter.Requester) (*openai.ChatCompletionNewParams, error) {
	r := requester.ToRequest()
	contents := make([]openai.ChatCompletionMessageParamUnion, 0, len(r.Messages))

	if r.ThreadId != nil {
		contents = append(contents, p.history.Load(r.ThreadId)...)
	}

	model, ok := lo.Coalesce(r.Model, p.model, lo.ToPtr(llm.DefaultModel()))
	if !ok {
		return nil, errors.New("no model was configured")
	}

	cfg := openai.ChatCompletionNewParams{
		Model:    *model,
		Messages: contents,
	}

	if r.MaxCandidates != nil {
		cfg.N = openai.Int(int64(*r.MaxCandidates))
	}
	if r.MaxTokens != nil {
		cfg.MaxTokens = openai.Int(int64(*r.MaxTokens))
	}
	if r.Temperature != nil {
		cfg.Temperature = openai.Float(*r.Temperature)
	}
	if r.TopP != nil {
		cfg.TopP = openai.Float(*r.TopP)
	}

	if r.ResponseSchema != nil {
		cfg.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        r.SchemaName,
					Description: openai.String(r.SchemaDescription),
					Schema:      lo.CoalesceOrEmpty(r.SchemaOverride, r.ResponseSchema),
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
			if seeker, ok := part.(io.ReadSeeker); ok {
				if _, err := seeker.Seek(0, io.SeekStart); err != nil {
					return nil, err
				}
			}

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

		if r.ThreadId != nil && !r.SkipSaveInput {
			p.history.Save(r.ThreadId, content)
		}

		cfg.Messages = append(cfg.Messages, content)
	}

	return &cfg, nil
}

func (p *OpenAi) adaptResponse(llm internal.Adapter, response *openai.ChatCompletion, requester llmadapter.Requester) (*llmadapter.InnerResponse, error) {
	resp := llmadapter.InnerResponse{
		Id:         response.ID,
		Model:      response.Model,
		Candidates: make([]llmadapter.ResponseCandidate, len(response.Choices)),
		Created:    time.Unix(response.Created, 0),
	}

	for idx, candidate := range response.Choices {
		var finishReason llmadapter.FinishReason

		switch candidate.FinishReason {
		case "stop":
			finishReason = llmadapter.FinishReasonStop
		case "length":
			finishReason = llmadapter.FinishReasonMaxTokens
		case "content_filter":
			finishReason = llmadapter.FinishReasonContentFilter
		default:
			finishReason = llmadapter.FinishReason(candidate.FinishReason)
		}

		toolCalls := make([]llmadapter.ResponseToolCall, len(candidate.Message.ToolCalls))

		for idx, toolCall := range candidate.Message.ToolCalls {
			toolCalls[idx] = llmadapter.ResponseToolCall{
				Id:         toolCall.ID,
				Name:       toolCall.Function.Name,
				Parameters: []byte(toolCall.Function.Arguments),
			}
		}

		resp.Candidates[idx] = llmadapter.ResponseCandidate{
			Text:         candidate.Message.Content,
			ToolCalls:    toolCalls,
			FinishReason: finishReason,
			SelectCandidate: func() {
				req := requester.ToRequest()

				if req.ThreadId != nil && !req.SkipSaveOutput {
					msg := openai.ChatCompletionMessageParamUnion{
						OfAssistant: &openai.ChatCompletionAssistantMessageParam{
							ToolCalls: candidate.Message.ToParam().GetToolCalls(),
							Content: openai.ChatCompletionAssistantMessageParamContentUnion{
								OfString: openai.String(candidate.Message.Content),
							},
						},
					}

					p.history.Save(req.ThreadId, msg)
				}
			},
		}
	}

	return &resp, nil
}

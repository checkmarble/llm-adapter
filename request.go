package llmadapter

import (
	"context"
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/invopop/jsonschema"
	"github.com/samber/lo"
)

type (
	MessageRole int
	MessageType string
)

const (
	RoleSystem MessageRole = iota
	RoleUser
	RoleAi
	RoleTool
)

const (
	TypeText = "text/plain"
)

type InnerRequest struct {
	Model          *string
	Messages       []Message
	ResponseSchema *Schema
	Tools          map[string]Tool
}

type Message struct {
	Type  MessageType
	Role  MessageRole
	Parts []io.Reader

	Tool *ResponseToolCall
}

type TypedRequest[T any] struct {
	InnerRequest

	respondsTo *ResponseCandidate
	err        error
}

func NewUntypedRequest() TypedRequest[string] {
	return TypedRequest[string]{
		InnerRequest: InnerRequest{
			Tools: make(map[string]Tool),
		},
	}
}

func NewRequest[T any]() TypedRequest[T] {
	r := InnerRequest{}

	switch any(*new(T)).(type) {
	case string:
	default:
		r.ResponseSchema = lo.ToPtr(GenerateSchema[T]("", ""))
	}

	return TypedRequest[T]{
		InnerRequest: r,
	}
}

func (r TypedRequest[T]) Do(ctx context.Context, llm *LlmAdapter) (*TypedResponse[T], error) {
	if r.err != nil {
		return nil, r.err
	}

	resp, err := llm.ChatCompletion(ctx, r.InnerRequest)
	if err != nil {
		return nil, err
	}

	return &TypedResponse[T]{*resp}, nil
}

func (r TypedRequest[T]) FromCandidate(c Candidater, idx int) TypedRequest[T] {
	candidate, err := c.Candidate(idx)
	if err != nil {
		r.err = errors.CombineErrors(r.err, err)
		return r
	}

	r.respondsTo = candidate
	candidate.SelectCandidateFunc()

	return r
}

func (r TypedRequest[T]) WithModel(model string) TypedRequest[T] {
	r.Model = &model

	return r
}

func (r TypedRequest[T]) WithInstruction(parts ...string) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type: TypeText,
		Role: RoleSystem,
		Parts: lo.Map(parts, func(p string, _ int) io.Reader {
			return strings.NewReader(p)
		}),
	})

	return r
}

func (r TypedRequest[T]) WithInstructionReader(parts ...io.Reader) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleSystem,
		Parts: parts,
	})

	return r
}

func (r TypedRequest[T]) WithText(role MessageRole, parts ...string) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type: TypeText,
		Role: role,
		Parts: lo.Map(parts, func(p string, _ int) io.Reader {
			return strings.NewReader(p)
		}),
	})

	return r
}

func (r TypedRequest[T]) WithTextReader(role MessageRole, parts ...io.Reader) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  role,
		Parts: parts,
	})

	return r
}

func (r TypedRequest[T]) WithTools(tools ...Tool) TypedRequest[T] {
	for _, tool := range tools {
		r.Tools[tool.Name] = tool
	}

	return r
}

func (r TypedRequest[T]) withToolResponse(tool ResponseToolCall, parts string) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleTool,
		Parts: []io.Reader{strings.NewReader(parts)},
		Tool:  &tool,
	})

	return r
}

func (r TypedRequest[T]) WithToolExecution(tools ...Tool) TypedRequest[T] {
	if r.respondsTo == nil {
		r.err = errors.CombineErrors(r.err, errors.New("cannot execute tool %s without selecting a response candidate, call FromCandidate() first"))
		return r
	}

	for _, tool := range tools {
		r = r.WithTools(tool)
	}

	for _, toolCall := range r.respondsTo.ToolCalls {
		tool, ok := r.Tools[toolCall.Name]

		if !ok {
			r.err = errors.Wrapf(r.err, "no tool was registered for response to tool '%s'", toolCall.Name)
			return r
		}

		resp, err := tool.Call(toolCall.Parameters)
		if err != nil {
			r.err = errors.CombineErrors(r.err, err)
			return r
		}

		r = r.withToolResponse(toolCall, resp)
	}

	return r
}

type Schema struct {
	Name        string
	Description string
	Schema      jsonschema.Schema
}

func GenerateSchema[S any](name string, description string) Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(new(S))

	return Schema{
		Name:        name,
		Description: description,
		Schema:      *schema,
	}
}

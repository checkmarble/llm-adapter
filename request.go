package llmadapter

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/invopop/jsonschema"
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
	schema, _ := GenerateSchema[T]("", "")

	r := InnerRequest{}

	switch any(*new(T)).(type) {
	case string:
	default:
		r.ResponseSchema = schema
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

func (r TypedRequest[T]) FromCandidate(resp Response, candidate int) TypedRequest[T] {
	if candidate > len(resp.Candidates)-1 {
		r.err = errors.Join(r.err, errors.New("selected candidate does not exist"))
		return r
	}

	r.respondsTo = &resp.Candidates[candidate]
	resp.Candidates[candidate].SelectCandidateFunc()

	return r
}

func (r TypedRequest[T]) WithModel(model string) TypedRequest[T] {
	r.Model = &model

	return r
}

func (r TypedRequest[T]) WithSystemInstruction(parts ...io.Reader) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleSystem,
		Parts: parts,
	})

	return r
}

func (r TypedRequest[T]) WithText(role MessageRole, parts ...io.Reader) TypedRequest[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  role,
		Parts: parts,
	})

	return r
}

func (r TypedRequest[T]) WithTool(tool *Tool) TypedRequest[T] {
	r.Tools[tool.Name] = *tool

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

func (r TypedRequest[T]) WithToolExecution(tools ...*Tool) TypedRequest[T] {
	if r.respondsTo == nil {
		r.err = errors.Join(r.err, errors.New("cannot execute tool without selecting a response candidate, call FromCandidate() first"))
		return r
	}
	for _, tool := range tools {
		if tool == nil {
			r.err = errors.Join(r.err, errors.New("unknown tool"))
			return r
		}

		r = r.WithTool(tool)
	}

	for _, toolCall := range r.respondsTo.ToolCalls {
		tool := r.Tools[toolCall.Name]

		resp, err := tool.Call(toolCall.Parameters)
		if err != nil {
			r.err = errors.Join(err)
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

func GenerateSchema[S any](name string, description string) (*Schema, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := reflector.Reflect(new(S))
	if schema == nil {
		return nil, errors.New("invalid response schema")
	}

	return &Schema{
		Name:        name,
		Description: description,
		Schema:      *schema,
	}, nil
}

package llmadapter

import (
	"context"
	"errors"
	"io"

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
)

const (
	TypeText = "text/plain"
)

type InnerRequest struct {
	Model          *string
	Messages       []Message
	ResponseSchema *Schema
}

type Message struct {
	Type  MessageType
	Role  MessageRole
	Parts []io.Reader
}

type TypedRequest[T any] struct {
	InnerRequest
}

func NewUntypedRequest() TypedRequest[string] {
	return TypedRequest[string]{
		InnerRequest: InnerRequest{},
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
	resp, err := llm.ChatCompletion(ctx, r.InnerRequest)
	if err != nil {
		return nil, err
	}

	return &TypedResponse[T]{*resp}, nil
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

func (r TypedRequest[T]) WithResponseSchema(schema Schema) TypedRequest[T] {
	r.ResponseSchema = &schema

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

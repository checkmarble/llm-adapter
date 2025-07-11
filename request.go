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
	MessageType int
)

const (
	RoleSystem MessageRole = iota
	RoleUser
	RoleAi
	RoleTool
)

const (
	TypeText MessageType = iota
)

type Message struct {
	Type  MessageType
	Role  MessageRole
	Parts []io.Reader

	Tool *ResponseToolCall
}

// innerRequest represents the actual request to be sent to the
// provider, before being adapted for it.
type innerRequest struct {
	Model          *string
	Messages       []Message
	Grounding      bool
	ResponseSchema *schema
	Tools          map[string]Tool
}

// Request represent a request to be sent the an LLM provider, in the
// context of the current conversation.
//
// It contains an `innerRequest` built by the caller, but also optionally
// tracks which candidate it responds to, in order to link tool responses
// to their corresponding tool calls.
//
// It is generic in T which it will use to unmarshal the reponse into a
// typed struct.
type Request[T any] struct {
	innerRequest

	provider   *string
	respondsTo *ResponseCandidate
	err        error
}

// NewUntypedRequest is a helper method to create a `Request` which will be
// a raw string, without unmarshalling the response into a struct.
func NewUntypedRequest() Request[string] {
	return Request[string]{
		innerRequest: innerRequest{
			Tools: make(map[string]Tool),
		},
	}
}

// NewRequest creates a builder to craft a reques to sent to an LLM provider.
//
// It provides a series of methods to chain-call in order to add context and prompts.
//
// It is generic in T, which will be used to generate a JSONSchema to be used as
// a response schema in the request. See [this](https://github.com/invopop/jsonschema)
// for more information about how to write the structs.
//
// Example usage:
//
//	resp, err := llmadapter.NewRequest[Output]().
//		WithText(llmadapter.RoleUser, "How are you today?").
//		Do(ctx, llm)
func NewRequest[T any]() Request[T] {
	r := innerRequest{
		Tools: make(map[string]Tool),
	}

	switch any(*new(T)).(type) {
	case string:
	default:
		r.ResponseSchema = lo.ToPtr(generateSchema[T]("", ""))
	}

	return Request[T]{
		innerRequest: r,
	}
}

// Do executes a built request on the configured LLM provider.
func (r Request[T]) Do(ctx context.Context, llm *LlmAdapter) (*TypedResponse[T], error) {
	if r.err != nil {
		return nil, r.err
	}
	if llm.defaultProvider == nil {
		return nil, errors.New("no provider was configured")
	}

	provider := llm.defaultProvider

	if r.provider != nil {
		p, ok := llm.providers[*r.provider]
		if !ok {
			return nil, errors.Newf("unknown provider '%s'", *r.provider)
		}

		provider = p
	}

	resp, err := provider.ChatCompletion(ctx, llm, r)
	if err != nil {
		return nil, err
	}

	return &TypedResponse[T]{*resp}, nil
}

func (r Request[T]) WithProvider(name string) Request[T] {
	r.provider = &name

	return r
}

// FromCandidate selects a candidate/choice from a previous response as the base
// for this Request.
//
// Selecting a candidate will have two effects:
//   - Adding the candidate to the history (if it is enabled).
//   - Using this response tool calls as a basis for tool responses, if applicable.
func (r Request[T]) FromCandidate(c Candidater, idx int) Request[T] {
	candidate, err := c.Candidate(idx)
	if err != nil {
		r.err = errors.CombineErrors(r.err, err)
		return r
	}

	r.respondsTo = candidate
	candidate.SelectCandidate()

	return r
}

// WithModel overrides the model used for this specific request.
//
// If not provided, the default model set on the adapter will be used.
func (r Request[T]) WithModel(model string) Request[T] {
	r.Model = &model

	return r
}

// WithInstruction adds a system prompt to the request.
//
// Note that if the adapter is configured to save history, this need only be
// added on the first request sent to the provider.
func (r Request[T]) WithInstruction(parts ...string) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type: TypeText,
		Role: RoleSystem,
		Parts: lo.Map(parts, func(p string, _ int) io.Reader {
			return strings.NewReader(p)
		}),
	})

	return r
}

// WithInstructionReader adds a system prompt read from an `io.Reader`.
func (r Request[T]) WithInstructionReader(parts ...io.Reader) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleSystem,
		Parts: parts,
	})

	return r
}

// WithText adds a text message to the Request.
//
// Each provided `string` will be added as a discrete `part` in the message. The
// message will be declared as text content.
func (r Request[T]) WithText(role MessageRole, parts ...string) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type: TypeText,
		Role: role,
		Parts: lo.Map(parts, func(p string, _ int) io.Reader {
			return strings.NewReader(p)
		}),
	})

	return r
}

// WithTextReader adds a message to the Request read from an `io.Reader`
func (r Request[T]) WithTextReader(role MessageRole, parts ...io.Reader) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  role,
		Parts: parts,
	})

	return r
}

// WithGrounding turns on public grounding for a specific request.
//
// It will only have an effect on provider supporting public methods to ground
// requests.
func (r Request[T]) WithGrounding() Request[T] {
	r.Grounding = true

	return r
}

// WithTools adds tool definitions to the request.
//
// See `tools.go` for more information about how to define tools.
func (r Request[T]) WithTools(tools ...Tool) Request[T] {
	for _, tool := range tools {
		r.Tools[tool.Name] = tool
	}

	return r
}

func (r Request[T]) withToolResponse(tool ResponseToolCall, parts string) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleTool,
		Parts: []io.Reader{strings.NewReader(parts)},
		Tool:  &tool,
	})

	return r
}

// WithToolExecution executes the requested tools and add their output to the Request.
//
// It will also take care of adding the matching tool definitions to the Request, so there
// is not need to also call `WithTool`.
//
// Note that this requires that a candidate from the previous reponse was selected by
// calling `FromCandidate()` before this function.
func (r Request[T]) WithToolExecution(tools ...Tool) Request[T] {
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
			r.err = errors.CombineErrors(r.err, errors.Newf("no tool was registered for response to tool '%s'", toolCall.Name))
			return r
		}

		resp, err := tool.call(toolCall.Parameters)
		if err != nil {
			r.err = errors.CombineErrors(r.err, err)
			return r
		}

		r = r.withToolResponse(toolCall, resp)
	}

	return r
}

type schema struct {
	Name        string
	Description string
	Schema      jsonschema.Schema
}

func generateSchema[S any](name string, description string) schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	jsonSchema := reflector.Reflect(new(S))

	return schema{
		Name:        name,
		Description: description,
		Schema:      *jsonSchema,
	}
}

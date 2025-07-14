package llmadapter

import (
	"context"
	"io"
	"reflect"
	"strings"

	"github.com/checkmarble/marble-llm-adapter/internal"
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

// Requester represents something that can be turned into a request.
//
// Used internally to abstract over request types across packages.
type Requester interface {
	// ToRequest unwraps the actual request.
	ToRequest() innerRequest
	// ProviderRequestOptions extracts the provider-specific configuration
	// options for a given provider. This is called from each provider to
	// retrieve its specific configuration in a type-safe manner.
	ProviderRequestOptions(provider Llm) internal.ProviderRequestOptions
}

// Message is an abstraction over a "prompt".
type Message struct {
	// Type is the binary representation of the message
	Type MessageType
	// Role represent "who" (or "what") composed a message. Note that all
	// provider will not support all of the roles, but must still account for
	// them.
	Role MessageRole
	// Parts are subdivision of a specific message.
	Parts []io.Reader

	// Tool is an instruction from a tool function to be called. This only makes
	// sense in response messages.
	Tool *ResponseToolCall
}

// innerRequest represents the actual request to be sent to the provider, before
// being adapted for it.
type innerRequest struct {
	Model          *string
	Messages       []Message
	ResponseSchema *jsonschema.Schema
	Tools          map[string]internal.Tool

	MaxTokens     *int
	MaxCandidates *int
	Temperature   *float64
	TopP          *float64

	ProviderOptions map[reflect.Type]internal.ProviderRequestOptions
}

// Request represent a request to be sent the a provider, in the context of the
// current conversation.
//
// It contains an `innerRequest` built by the caller, but also optionally tracks
// which candidate it responds to, in order to link tool responses to their
// corresponding tool calls.
//
// It is generic in T which it will use to unmarshal the reponse into a typed
// struct.
type Request[T any] struct {
	innerRequest

	provider   *string
	respondsTo *ResponseCandidate
	err        error
}

// NewUntypedRequest is a helper method to create a `Request` which will be a
// raw string, without unmarshalling the response into a struct.
func NewUntypedRequest() Request[string] {
	return Request[string]{
		innerRequest: innerRequest{
			Tools:           make(map[string]internal.Tool),
			ProviderOptions: make(map[reflect.Type]internal.ProviderRequestOptions),
		},
	}
}

// NewRequest creates a builder to craft a request to sent to an LLM provider.
//
// It provides a series of methods to chain-call in order to add context,
// prompts and configuration.
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
		Tools:           make(map[string]internal.Tool),
		ProviderOptions: make(map[reflect.Type]internal.ProviderRequestOptions),
	}

	switch any(*new(T)).(type) {
	case string:
	default:
		r.ResponseSchema = lo.ToPtr(internal.GenerateSchema[T]())
	}

	return Request[T]{
		innerRequest: r,
	}
}

// Do executes a built request on the configured provider.
//
// It will return a response generic over the configured typed on the Request,
// or an error.
func (r Request[T]) Do(ctx context.Context, llm *LlmAdapter) (*Response[T], error) {
	if r.err != nil {
		return nil, r.err
	}

	provider, err := llm.GetProvider(r.provider)
	if err != nil {
		return nil, err
	}

	resp, err := provider.ChatCompletion(ctx, llm, r)
	if err != nil {
		return nil, err
	}

	return &Response[T]{*resp}, nil
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
//
// Example usage:
//
//	resp, err := llmadapter.NewRequest[Output]().
//		FromCandidate(previousResp, 0).
//		WithText(llmadapter.RoleUser, "How are you today?").
//		Do(ctx, llm)
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
// If not provided, the default model set on the provider, then the adapter will
// be used.
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

// WithInstructionReader adds a system prompt read from an io.Reader.
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

// WithTextReader adds a message to the Request read from an io.Reader.
func (r Request[T]) WithTextReader(role MessageRole, parts ...io.Reader) Request[T] {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  role,
		Parts: parts,
	})

	return r
}

// WithTools adds tool definitions to the request.
//
// Tools are represented as a type-safe function taking its configuration as
// input, and return a string and an error. The JSONSchema sent to the provider
// will be generated from the input type.
//
// Example usage:
//
//	resp, err := llmadapter.NewRequest[Output]().
//		WithText(llmadapter.RoleUser, "How are you today?").
//		WithTool(llmadapter.NewTool[WeatherParams]("get_weather", "Get weather at location", llmadapter.Function(func(args WeatherParams) (string, error) {
//			return "Good weather!", nil
//		})).
//		Do(ctx, llm)
func (r Request[T]) WithTools(tools ...internal.Tool) Request[T] {
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

// WithToolExecution executes the requested tools and add their output to the
// Request.
//
// It will also take care of adding the matching tool definitions to the
// Request, so there is not need to also call `WithTool`.
//
// Note that this requires that a candidate from the previous reponse was
// selected by calling `FromCandidate()` before this function, to determine
// which function the provider asked to be called.
func (r Request[T]) WithToolExecution(tools ...internal.Tool) Request[T] {
	if r.respondsTo == nil {
		r.err = errors.CombineErrors(r.err, errors.Newf("cannot execute tools without selecting a response candidate, call FromCandidate() first"))
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

		resp, err := tool.Call(toolCall.Parameters)
		if err != nil {
			r.err = errors.CombineErrors(r.err, err)
			return r
		}

		r = r.withToolResponse(toolCall, resp)
	}

	return r
}

// WithProviderOptions set provier-specific options.
//
// Some options are not going to be supported by all providers, so they will
// usually defined a type representing options specific to them. This function
// allows to define those. One set of option can be defined by provider type.
func (r Request[T]) WithProviderOptions(opts internal.ProviderRequestOptions) Request[T] {
	r.ProviderOptions[reflect.TypeOf(opts)] = opts

	return r
}

// WithMaxTokens limits how many token a provider can emit for its completion.
func (r Request[T]) WithMaxTokens(tokens int) Request[T] {
	r.MaxTokens = &tokens

	return r
}

// WithMaxCandidates limits how many candidate responses the provider is able to provide.
//
// Most providers default to 1 for this value.
func (r Request[T]) WithMaxCandidates(candidates int) Request[T] {
	r.MaxCandidates = &candidates

	return r
}

// WithTemperature sets custom temperature value to be used.
//
// Default value depends on the model.
func (r Request[T]) WithTemperature(temp float64) Request[T] {
	r.Temperature = &temp

	return r
}

// WithTopP sets the `top_p` parameter.
func (r Request[T]) WithTopP(topp float64) Request[T] {
	r.TopP = &topp

	return r
}

// Request[T] implementation of Requester.

func (r Request[T]) ToRequest() innerRequest {
	return r.innerRequest
}

func (r Request[T]) ProviderRequestOptions(provider Llm) internal.ProviderRequestOptions {
	var providerOpts internal.ProviderRequestOptions

	if opts, ok := r.ProviderOptions[provider.RequestOptionsType()]; ok {
		providerOpts = opts
	}

	return providerOpts
}

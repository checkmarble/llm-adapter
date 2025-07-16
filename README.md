# [WIP] LLM Adapter

Type-safe wrapper adapter around various LLM providers.

## Usage

### Basic setup

LLM Adapter is used by setting up an instance of it with _at least_ one provider. An adapter can be configured with more (named) providers, which can be selected by their names when sending requests.

```go
gpt, err := openai.New(openai.WithApiKey("..."))
gemini, err := aistudio.New()

llm, err := llmadapter.New(
	llmadapter.WithDefaultProvider(gpt),
	llmadapter.WithAdapter("gemini", gemini),
	llmadapter.WithDefaultModel("gpt-4"),
)
```

An adapter always has a default provider that will be used when no specific provider is specified on a request. The default provider is either the one added with `WithDefaultProvider()`, or the first named provider given.

Each provider _may_ offer some options for customization that are specific to it. Refer to each provider's package to know which options they offer.

### Requests

#### Typed output

Requests are built through a series of chainable methods determining its content and behavior. A request is typed with the Go type of the expected response. When a type is given, the appropriate response format will be set on the request so the provider responds with a JSON string of the appropriate schema.

The struct tags one can add to the given type are explained in [this repository](https://github.com/invopop/jsonschema).

```go
type Output struct {
	LightColor string `json:"text" jsonschema_description:"Color of the traffic light" jsonschema:"enum=red,enum=yellow=enum=red"`
}

req, err := llmadapter.NewRequest[Output]()
req, err := llmadapter.NewUntypedRequest() // Equivalent to `NewRequest[string]()`
````

If you wish for your response to be serialized into a type that cannot be represented as a static struct (for example, if you build your types dynamically), you can specify the schema yourself with `OverrideResponseSchema()`. Note that this schema still requires to be unserializable into the provided type.

```go

props := jsonschema.NewProperties()
props.Set("reply", &jsonschema.Schema{
	Type: "string",
	Description: "Your response to my question",
})

schema := jsonschema.Schema{
	Type: "object",
	Properties: props,
}

req, err := NewRequest[map[string]string]().
	OverrideResponseSchema(schema)
````

#### Provider and model selection

Both provider and model used in a request can be selected with the builder methods `WithProvider()` and `WithModel()`. If not provided:

 - The default provider will be used
 - The model of the request will be used if set, or the default model for the provider if set, or the default model on the adapter

#### Prompting

Adding prompts is performed in a provider-agnostic way through a series of builder method on `Request[T]` and offer a variety of input media. So far, only text input are supported.

```go
req.
	WithInstruction("system prompt").
	WithInstructionReader(strings.NewReader("system prompt")).
	WithInstructionFile("/etc/prompt.md").
	WithText(llmadapter.RoleUser, "user prompt").
	WithTextReader(llmadapter.RoleUser, strings.NewReader("user prompt")).
	WithJson(llmadapter.RoleUser, typ). // Any JSON-serializable type
	WithSerializable(llmadapter.RoleUser, llmadapter.Serializers.Json, typ) // Use a decoder implementing llmadapter.Serializer
````

#### Executing

Executing a request is done by calling the `Do()` method on a request. A response will contain generic information about the response, and one or more candidate responses (depending on the configuration of the request).

To obtain the typed, deserialized output of one of the candidate, use `resp.Get(idx)` (`idx` being the index of the candidate).

```go
resp, err := req.Do(ctx, llm)
output, err := resp.Get(0)
````

A few utilities are available to run multiple requests at the same time:

 - `llmadapter.All[T](context.Context, *llmadapter.LlmAdapter, reqs ...Request[T])` can be used to fire several requests at once, wait for all of them to return and get a slice of results.
 - `llmadapter.Race[T](context.Context, *llmadapter.LlmAdapter, reqs ...Request[T])` can be used to fire several requests at once, return the first successful response, and cancel the others.

Note that cancelled requests will still incur cost on most providers.

#### Chaining

To conduct a conversation, one candidate must be selected as the basis for the next request.

```go
resp1, err := req.Do(ctx, llm)
resp2, err := req.FromCandidate(resp1, 0).Do(ctx, llm)
````

#### History

By default, every request will be sent with a blank context. To opt into history accumulation (building a context through the conversation), one can use `threads`. By starting a threads in one request, and then re-using that same thread in subsequent requests, inputs and outputs will be accumulated and sent with every request.

Each thread is represented by an opaque, non-copyable `*ThreadId` which is associated with the provider that created it. A thread cannot be shared across providers.

**Warning:** A `ThreadId` must not be copied, which is why it should always be handled as a pointer. Go will emit warnings if it is copied anywhere.

```go
resp1, err := req.CreateThread().Do(ctx, llm)
resp2, err := req.InThread(resp1.ThreadId).Do(ctx, llm)
````

To send a new request with a clear history, either send a request without using a thread method, create a new thread, or clear the thread with `resp.ThreadId.Clear()`. It can be copied with `resp.ThreadId.Copy()`.

When using thread, by default, both inputs and outputs are saved. To opt out of storing one or both of those, you can chain the `SkipSaveInput()` or `SkipSaveOutput()` on the request.

Note that starting a response from a previous candidate automatically adds that response to the relevant thread history.

Threads should be closed after you are done using them to clean associated resources. We recomment deferring a call to `(*ThreadId).Close()` after you create it. If you do not, threads will live on until the whole adapter is garbage collected.

#### Tool calling

Tools can be defined in a type-safe manner by using the `NewTool` function and refering to it in various requests. A function consists of a name, a description and a callback taking an arbitrary type as argument and returning `(string, error)`.

```go
type WeatherToolParams struct {
	Location string `json:"location" jsonschema_description:"The location for which to retrieve the weather forecast"`
}

weatherTool := llmadapter.NewTool[WeatherToolParams](
	"get_weather_in_location",
	"Get a weather forecast in a given location",
	llmadapter.Function(func(p WeatherToolParams) (string, error) {
		return "Weather is going to be very rainy with chance of thunderstorms", nil
	}),
)

resp1, err := llmadapter.NewUntypedRequest().CreateThread().
	WithText(llmadapter.RoleUser, "Tell me the weather in Paris.").
	WithTools(weatherTool).
	Do(ctx, llm)

resp2, err := llmadapter.NewUntypedRequest().FromCandidate(resp1, 0).
	WithToolExecution(weatherTool).
	Do(ctx, llm)
```

A lot is happening here:

 - A tool is defined, taking a `WeatherToolParams` as argument. This type will be serialized into a JSON schema to instruct the LLM how to communicate arguments.
 - A request requiring a tool is sent, in a thread.
 - A second request selects a previous candidate (joining its thread), and executes any requested function, appending the output to the request.

Tool calling only works on request that are part of a thread, since providing history is required.

Note that `WithToolExecution` will fail if a candidate was not selected **beforehand** or if the previous response is not part of a thread.

## Example

See the executables in `examples/` for more complete examples.

```go
type Output struct {
	Reply  string `json:"reply" jsonschema_description:"The response you want to give me"`
	Random int    `json:"random" jsonschema_description:"A random number you must generate between 100 and 200"`
}

func main() {
	ctx := context.Background()
	systemPrompt, _ := os.Open("../prompts/system.txt")

	provider, _ := aistudio.New(
		aistudio.WithBackend(genai.BackendVertexAI),
		aistudio.WithProject(os.Getenv("GOOGLE_CLOUD_PROJECT")),
		aistudio.WithLocation("europe-west1"),
	)

	llm, _ := llmadapter.New(
		llmadapter.WithDefaultProvider(provider),
		llmadapter.WithDefaultModel("gemini-2.5-flash"),
		llmadapter.WithApiKey(os.Getenv("LLM_API_KEY")),
	)

	resp, _ := llmadapter.NewRequest[Output]().
		WithInstructionReader(systemPrompt).
		WithText(llmadapter.RoleUser, "Hello, my name is Antoine!").
		Do(ctx, llm)

	obj, _ := resp.Get(0)

	fmt.Println("Reply:", obj.Reply)
	fmt.Println("Random number:", obj.Random)
}
```

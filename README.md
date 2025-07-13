# [WIP] LLM Adapter

Type-safe wrapper adapter around LLM providers.

## Usage

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

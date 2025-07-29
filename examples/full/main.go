package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmadapter "github.com/checkmarble/llm-adapter"
	"github.com/checkmarble/llm-adapter/llms/aistudio"
	"google.golang.org/genai"
)

type Output struct {
	Reply  string `json:"reply" jsonschema_description:"The response you want to give me"`
	Random int    `json:"random" jsonschema_description:"A random number you must generate between 100 and 200"`
}

func main() {
	ctx := context.Background()

	provider, err := aistudio.New(
		aistudio.WithBackend(genai.BackendVertexAI),
		aistudio.WithProject(os.Getenv("GOOGLE_CLOUD_PROJECT")),
		aistudio.WithLocation("europe-west1"),
		aistudio.WithApiKey(os.Getenv("LLM_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	llm, err := llmadapter.New(
		llmadapter.WithProvider("vertex", provider),
		llmadapter.WithDefaultModel("gemini-2.5-pro"),
	)

	if err != nil {
		log.Fatal(err)
	}

	resp1, err := llmadapter.NewRequest[Output]().
		CreateThread().
		WithInstructionFiles("prompts/system.txt").
		WithText(llmadapter.RoleUser, "Hello, my name is Bob!").
		Do(ctx, llm)

	if err != nil {
		log.Fatal(err)
	}

	defer resp1.ThreadId.Close()

	out, _ := resp1.Get(0)

	fmt.Println("Reply:", out.Reply, "Random:", out.Random)

	resp2, err := llmadapter.NewUntypedRequest().
		FromCandidate(resp1, 0).
		WithText(llmadapter.RoleUser, "Do you remember what my name is? Also, append your previous response.").
		Do(ctx, llm)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp2.Get(0))

	resp1.ThreadId.Clear()

	type WeatherToolParams struct {
		Location string `json:"location" jsonschema_description:"The location for which to retrieve the weather forecast"`
	}

	weatherTool := llmadapter.NewTool[WeatherToolParams]("get_weather_in_location", "Get a weather forecast in a given location", llmadapter.Function(func(p WeatherToolParams) (string, error) {
		return "Weather is going to be very rainy with chance of thunderstorms", nil
	}))

	if err != nil {
		log.Fatal(weatherTool)
	}

	resp3, err := llmadapter.NewUntypedRequest().
		CreateThread().
		FromCandidate(resp2, 0).
		WithText(llmadapter.RoleUser, "Tell me the weather in Paris.").
		WithTools(weatherTool).
		Do(ctx, llm)

	if err != nil {
		log.Fatal(err)
	}

	defer resp3.ThreadId.Close()

	resp4, err := llmadapter.NewUntypedRequest().
		FromCandidate(resp3, 0).
		WithToolExecution(weatherTool).
		Do(ctx, llm)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp4.Get(0))
}

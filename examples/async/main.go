package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/aistudio"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))
	gemini, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI), aistudio.WithApiKey(os.Getenv("LLM_API_KEY")))

	llm, _ := llmadapter.New(
		llmadapter.WithProvider("ollama", ollama),
		llmadapter.WithProvider("gemini", gemini),
		llmadapter.WithDefaultModel("gemma3n:e4b"),
	)

	fmt.Println("All()")

	resps := llmadapter.All(
		ctx, llm,
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "How are you?"),
		llmadapter.NewUntypedRequest().WithProvider("gemini").WithModel("gemini-2.5-flash").WithText(llmadapter.RoleUser, "Compute 1 + 1."),
	)

	for _, resp := range resps {
		if resp.Error != nil {
			log.Fatal(resp.Error)
		}

		cand, _ := resp.Response.Get(0)

		fmt.Println(resp.Response.Model, "answered:")
		fmt.Println(cand)
		fmt.Println("-----")
	}

	fmt.Println("Race()")

	resp, err := llmadapter.Race(
		ctx, llm,
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "How are you?"),
		llmadapter.NewUntypedRequest().WithProvider("gemini").WithText(llmadapter.RoleUser, "Compute 1 + 1."),
	)

	if err != nil {
		log.Fatal(err)
	}

	cand, _ := resp.Get(0)

	fmt.Println(resp.Model, "answered:", cand)
}

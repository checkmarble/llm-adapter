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

	gemini, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI))
	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))

	llm, _ := llmadapter.New(
		llmadapter.WithProvider("vertex", gemini),
		llmadapter.WithProvider("ollama", ollama),
		llmadapter.WithDefaultModel("gemini-2.5-flash"),
		llmadapter.WithApiKey(os.Getenv("LLM_API_KEY")),
		llmadapter.WithSaveContext(),
	)

	ollamaResponse, err := llmadapter.NewUntypedRequest().WithProvider("ollama").WithModel("gemma3n:e4b").WithText(llmadapter.RoleUser, "How are you?").Do(ctx, llm)
	if err != nil {
		log.Fatal(err)
	}
	ollamaCandidate, _ := ollamaResponse.Candidate(0)

	vertexResponse, err := llmadapter.NewUntypedRequest().WithText(llmadapter.RoleUser, "How are you?").Do(ctx, llm)
	if err != nil {
		log.Fatal(err)
	}
	vertexCandidate, _ := vertexResponse.Candidate(0)

	fmt.Println(ollamaResponse.Model, "said", ollamaCandidate.Text)
	fmt.Println(vertexResponse.Model, "said", vertexCandidate.Text)
}

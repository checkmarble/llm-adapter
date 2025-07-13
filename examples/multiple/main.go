package main

import (
	"context"
	"fmt"
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

	ollamaResponse, _ := llmadapter.NewUntypedRequest().WithProvider("ollama").WithModel("gemma3n:e4b").WithText(llmadapter.RoleUser, "How are you?").Do(ctx, llm)
	ollamaCandidate, _ := ollamaResponse.Candidate(0)

	geminiResponse, _ := llmadapter.NewUntypedRequest().WithText(llmadapter.RoleUser, "How are you?").Do(ctx, llm)
	geminiCandidate, _ := geminiResponse.Candidate(0)

	fmt.Println(ollamaResponse.Model, "said", ollamaCandidate.Text)
	fmt.Println(geminiResponse.Model, "said", geminiCandidate.Text)
}

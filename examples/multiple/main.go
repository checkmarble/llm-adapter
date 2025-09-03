package main

import (
	"context"
	"fmt"
	"os"

	llmberjack "github.com/checkmarble/llmberjack"
	"github.com/checkmarble/llmberjack/llms/aistudio"
	"github.com/checkmarble/llmberjack/llms/openai"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	gemini, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI), aistudio.WithApiKey(os.Getenv("LLM_API_KEY")))
	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))

	llm, _ := llmberjack.New(
		llmberjack.WithProvider("vertex", gemini),
		llmberjack.WithProvider("ollama", ollama),
		llmberjack.WithDefaultModel("gemini-2.5-flash"),
	)

	ollamaResponse, _ := llmberjack.NewUntypedRequest().WithProvider("ollama").WithModel("gemma3n:e4b").WithText(llmberjack.RoleUser, "How are you?").Do(ctx, llm)
	ollamaCandidate, _ := ollamaResponse.Candidate(0)

	geminiResponse, _ := llmberjack.NewUntypedRequest().WithText(llmberjack.RoleUser, "How are you?").Do(ctx, llm)
	geminiCandidate, _ := geminiResponse.Candidate(0)

	fmt.Println(ollamaResponse.Model, "said", ollamaCandidate.Text)
	fmt.Println(geminiResponse.Model, "said", geminiCandidate.Text)
}

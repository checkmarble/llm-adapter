package main

import (
	"context"
	"fmt"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
)

func main() {
	ctx := context.Background()

	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))

	llm, _ := llmadapter.New(
		llmadapter.WithProvider("ollama", ollama),
		llmadapter.WithDefaultModel("gemma3n:e4b"),
		llmadapter.WithSaveContext(),
	)

	resp, _ := llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "How are you?").Do(ctx, llm)

	fmt.Println(resp)
}

package main

import (
	"context"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
	"github.com/k0kubun/pp/v3"
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

	pp.Println(resp)
}

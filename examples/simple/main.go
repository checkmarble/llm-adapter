package main

import (
	"context"
	"fmt"

	llmberjack "github.com/checkmarble/llmberjack"
	"github.com/checkmarble/llmberjack/llms/openai"
)

func main() {
	ctx := context.Background()

	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))

	llm, _ := llmberjack.New(
		llmberjack.WithProvider("ollama", ollama),
		llmberjack.WithDefaultModel("gemma3n:e4b"),
	)

	resp, _ := llmberjack.NewUntypedRequest().WithProvider("ollama").WithText(llmberjack.RoleUser, "How are you?").Do(ctx, llm)

	fmt.Println(resp)
}

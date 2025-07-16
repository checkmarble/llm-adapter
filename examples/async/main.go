package main

import (
	"context"
	"fmt"
	"log"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
)

func main() {
	ctx := context.Background()

	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))

	llm, _ := llmadapter.New(
		llmadapter.WithProvider("ollama", ollama),
		llmadapter.WithDefaultModel("gemma3n:e4b"),
	)

	resps := llmadapter.All(
		ctx, llm,
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "How are you?"),
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "Compute 1 + 1."),
	)

	for _, resp := range resps {
		if resp.Error != nil {
			log.Fatal(resp.Error)
		}

		cand, _ := resp.Response.Get(0)

		fmt.Println(cand)
	}

	resp := llmadapter.Race(
		ctx, llm,
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "How are you?"),
		llmadapter.NewUntypedRequest().WithProvider("ollama").WithText(llmadapter.RoleUser, "Compute 1 + 1."),
	)

	if resp.Error != nil {
		log.Fatal(resp.Error)
	}

	cand, _ := resp.Response.Get(0)

	fmt.Println(cand)
}

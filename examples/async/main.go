package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmberjack "github.com/checkmarble/llmberjack"
	"github.com/checkmarble/llmberjack/llms/aistudio"
	"github.com/checkmarble/llmberjack/llms/openai"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	ollama, _ := openai.New(openai.WithBaseUrl("http://localhost:11434/v1"))
	gemini, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI), aistudio.WithApiKey(os.Getenv("LLM_API_KEY")))

	llm, _ := llmberjack.New(
		llmberjack.WithProvider("ollama", ollama),
		llmberjack.WithProvider("gemini", gemini),
		llmberjack.WithDefaultModel("gemma3n:e4b"),
	)

	fmt.Println("All()")

	resps := llmberjack.All(
		ctx, llm,
		llmberjack.NewUntypedRequest().WithProvider("ollama").WithText(llmberjack.RoleUser, "How are you?"),
		llmberjack.NewUntypedRequest().WithProvider("gemini").WithModel("gemini-2.5-flash").WithText(llmberjack.RoleUser, "Compute 1 + 1."),
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

	resp, err := llmberjack.Race(
		ctx, llm,
		llmberjack.NewUntypedRequest().WithProvider("ollama").WithText(llmberjack.RoleUser, "How are you?"),
		llmberjack.NewUntypedRequest().WithProvider("gemini").WithText(llmberjack.RoleUser, "Compute 1 + 1."),
	)

	if err != nil {
		log.Fatal(err)
	}

	cand, _ := resp.Get(0)

	fmt.Println(resp.Model, "answered:", cand)
}

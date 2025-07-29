package main

import (
	"context"
	"fmt"
	"os"

	llmadapter "github.com/checkmarble/llm-adapter"
	"github.com/checkmarble/llm-adapter/llms/aistudio"
	"github.com/samber/lo"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	provider, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI), aistudio.WithApiKey(os.Getenv("LLM_API_KEY")))
	llm, _ := llmadapter.New(
		llmadapter.WithProvider("vertex", provider),
		llmadapter.WithDefaultModel("gemini-2.5-flash"),
	)

	resp, _ := llmadapter.NewUntypedRequest().
		WithProviderOptions(aistudio.RequestOptions{
			GoogleSearch: lo.ToPtr(true),
		}).
		WithText(llmadapter.RoleUser, "When was the Madleen ship stopped when trying to reach Gaza?").
		Do(ctx, llm)

	candidate, _ := resp.Candidate(0)

	fmt.Println(candidate.Text)

	fmt.Println("Searches:", candidate.Grounding.Searches)
	fmt.Println("Sources:")
	for _, src := range candidate.Grounding.Sources {
		fmt.Printf(" - %s\n", src.Domain)
	}
	fmt.Println("Snippets:")
	for _, snippet := range candidate.Grounding.Snippets {
		fmt.Printf(" - %s\n", snippet)
	}
}

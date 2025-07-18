package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/aistudio"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	provider, _ := aistudio.New(
		aistudio.WithBackend(genai.BackendVertexAI),
		aistudio.WithProject(os.Getenv("GOOGLE_CLOUD_PROJECT")),
		aistudio.WithLocation("europe-west1"),
		aistudio.WithApiKey(os.Getenv("LLM_API_KEY")),
		aistudio.WithBucket(os.Getenv("LLM_BATCH_BUCKET")),
	)

	llm, _ := llmadapter.New(
		llmadapter.WithProvider("vertex", provider),
		llmadapter.WithDefaultModel("gemini-2.5-flash"),
	)

	reqs := []llmadapter.Requester{
		llmadapter.NewUntypedRequest().WithProvider("vertex").WithId("how").WithText(llmadapter.RoleUser, "How are you?"),
		llmadapter.NewUntypedRequest().WithProvider("vertex").WithId("addition").WithText(llmadapter.RoleUser, "What is 1 + 1?"),
	}

	promise, err := llm.SubmitBatch(ctx, "vertex", reqs...)
	if err != nil {
		log.Fatal(err)
	}

	result := <-promise.Wait(ctx)

	if result.Error != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", result)
}

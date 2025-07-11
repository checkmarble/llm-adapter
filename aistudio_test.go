package llmadapter_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/aistudio"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

const aistudioResponse = `
{
	"modelVersion": "themodel",
    "candidates": [
        {
            "content": {
            	"role": "model",
            	"parts": [
             		{ "text": "{\"reply\":\"The JSON response from the provider.\"}" }
                ]
            }
        }
    ]
}
`

func TestGoogleAiRequest(t *testing.T) {
	defer gock.Off()

	type Output struct {
		Reply string `json:"reply" jsonschema_description:"Write your response here"`
	}

	type Args struct {
		Name string `json:"name" jsonschema_description:"My name"`
	}

	reader := strings.NewReader("text from reader")
	provider, _ := aistudio.New(aistudio.WithBackend(genai.BackendGeminiAPI))
	llm, _ := llmadapter.NewLlmAdapter(llmadapter.WithDefaultProvider(provider), llmadapter.WithApiKey("apikey"))

	req := llmadapter.NewRequest[Output]().
		WithModel("themodel").
		WithInstruction("system text").
		WithInstructionReader(reader).
		WithText(llmadapter.RoleUser, "user text").
		WithTools(llmadapter.NewTool[Args]("thetool", "Tool to get nothing", llmadapter.Function(func(Args) (string, error) {
			return "OK", nil
		}))).
		WithTextReader(llmadapter.RoleUser, reader)

	gock.New("https://generativelanguage.googleapis.com").
		Post("/v1beta/models/themodel:generateContent").
		MatchHeader("x-goog-api-key", "apikey").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, _ := io.ReadAll(req.Body)
			matched := bytes.Equal(bytes.TrimSpace(body), []byte(`{"contents":[{"parts":[{"text":"user text"}],"role":"user"},{"parts":[{}],"role":"user"}],"generationConfig":{"responseJsonSchema":{"$id":"https://github.com/checkmarble/marble-llm-adapter_test/output","$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,"properties":{"reply":{"description":"Write your response here","type":"string"}},"required":["reply"],"type":"object"},"responseMimeType":"application/json"},"systemInstruction":{"parts":[{"text":"text from reader"}],"role":"user"},"tools":[{"functionDeclarations":[{"description":"Tool to get nothing","name":"thetool","parametersJsonSchema":{"$id":"https://github.com/checkmarble/marble-llm-adapter_test/args","$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,"properties":{"name":{"description":"My name","type":"string"}},"required":["name"],"type":"object"}}]}]}`))

			return matched, nil
		}).
		Reply(http.StatusOK).
		SetHeader("content-type", "application/json").
		BodyString(aistudioResponse)

	resp, err := req.Do(t.Context(), llm)

	assert.False(t, gock.HasUnmatchedRequest())
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.NumCandidates())
	assert.Equal(t, "themodel", resp.Model)

	candidate, err := resp.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "The JSON response from the provider.", candidate.Reply)
}

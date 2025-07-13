package aistudio_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/aistudio"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"google.golang.org/genai"
)

const aistudioResponse = `{
	"responseId": "theid",
	"modelVersion": "themodel",
	"candidates": [
		{
			"finishReason": "STOP",
				"content": {
				"role": "model",
				"parts": [
					{ "text": "{\"reply\":\"The JSON response from the provider.\"}" }
				]
			}
		}
	],
	"createTime": "2025-07-13T16:20:00Z"
}`

func TestGoogleAiRequest(t *testing.T) {
	defer gock.Off()

	type Output struct {
		Reply string `json:"reply" jsonschema_description:"Write your response here"`
	}

	type Args struct {
		Name string `json:"name" jsonschema_description:"My name"`
	}

	httpClient := &http.Client{}
	provider, _ := aistudio.New(aistudio.WithBackend(genai.BackendVertexAI), aistudio.WithLocation("location"), aistudio.WithProject("project"))
	llm, err := llmadapter.New(llmadapter.WithDefaultProvider(provider), llmadapter.WithHttpClient(httpClient))

	req := llmadapter.NewRequest[Output]().
		WithModel("themodel").
		WithInstruction("system text").
		WithInstructionReader(strings.NewReader("text from reader")).
		WithText(llmadapter.RoleUser, "user text").
		WithTools(llmadapter.NewTool[Args]("thetool", "Tool to get nothing", llmadapter.Function(func(Args) (string, error) {
			return "OK", nil
		}))).
		WithTextReader(llmadapter.RoleUser, strings.NewReader("text from reader"))

	gock.InterceptClient(httpClient)

	gock.New("https://location-aiplatform.googleapis.com").
		Post("/v1beta1/projects/project/locations/location/publishers/google/models/themodel:generateContent").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, _ := io.ReadAll(req.Body)

			assert.EqualValues(t, 2, gjson.GetBytes(body, "systemInstruction.parts.#").Int())
			assert.Equal(t, "system text", gjson.GetBytes(body, "systemInstruction.parts.0.text").String())
			assert.Equal(t, "text from reader", gjson.GetBytes(body, "systemInstruction.parts.1.text").String())
			assert.Equal(t, "user", gjson.GetBytes(body, "systemInstruction.role").String())

			assert.EqualValues(t, 2, gjson.GetBytes(body, "contents.#").Int())
			assert.Equal(t, "user text", gjson.GetBytes(body, "contents.0.parts.0.text").String())
			assert.Equal(t, "user", gjson.GetBytes(body, "contents.0.role").String())
			assert.Equal(t, "text from reader", gjson.GetBytes(body, "contents.1.parts.0.text").String())
			assert.Equal(t, "user", gjson.GetBytes(body, "contents.1.role").String())

			assert.Equal(t, "object", gjson.GetBytes(body, "generationConfig.responseJsonSchema.type").String())
			assert.EqualValues(t, 1, gjson.GetBytes(body, "generationConfig.responseJsonSchema.properties|@keys|#").Int())
			assert.EqualValues(t, 1, gjson.GetBytes(body, "generationConfig.responseJsonSchema.required.#").Int())
			assert.Equal(t, "string", gjson.GetBytes(body, "generationConfig.responseJsonSchema.properties.reply.type").String())
			assert.Equal(t, "Write your response here", gjson.GetBytes(body, "generationConfig.responseJsonSchema.properties.reply.description").String())

			assert.EqualValues(t, 1, gjson.GetBytes(body, "tools.#").Int())
			assert.EqualValues(t, 1, gjson.GetBytes(body, "tools.0.functionDeclarations.#").Int())
			assert.Equal(t, "thetool", gjson.GetBytes(body, "tools.0.functionDeclarations.0.name").String())
			assert.Equal(t, "Tool to get nothing", gjson.GetBytes(body, "tools.0.functionDeclarations.0.description").String())
			assert.EqualValues(t, 1, gjson.GetBytes(body, "tools.0.functionDeclarations.0.parametersJsonSchema.properties|@keys|#").Int())
			assert.Equal(t, "string", gjson.GetBytes(body, "tools.0.functionDeclarations.0.parametersJsonSchema.properties.name.type").String())
			assert.Equal(t, "My name", gjson.GetBytes(body, "tools.0.functionDeclarations.0.parametersJsonSchema.properties.name.description").String())

			return true, nil
		}).
		Reply(http.StatusOK).
		SetHeader("content-type", "application/json").
		BodyString(aistudioResponse)

	resp, err := req.Do(t.Context(), llm)

	assert.False(t, gock.HasUnmatchedRequest())
	assert.Nil(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, "theid", resp.Id)
	assert.Equal(t, "themodel", resp.Model)
	assert.WithinDuration(t, time.Date(2025, 7, 13, 16, 20, 0, 0, time.UTC), resp.Created, 0)
	assert.Equal(t, 1, resp.NumCandidates())

	candidate, err := resp.Candidate(0)

	assert.Nil(t, err)
	assert.Equal(t, llmadapter.FinishReasonStop, candidate.FinishReason)

	output, err := resp.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "The JSON response from the provider.", output.Reply)
}

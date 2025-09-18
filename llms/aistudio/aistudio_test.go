package aistudio_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	llmberjack "github.com/checkmarble/llmberjack"
	"github.com/checkmarble/llmberjack/llms/aistudio"
	"github.com/h2non/gock"
	"github.com/samber/lo"
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
	llm, _ := llmberjack.New(llmberjack.WithDefaultProvider(provider), llmberjack.WithHttpClient(httpClient))

	req := llmberjack.NewRequest[Output]().
		WithModel("themodel").
		WithInstruction("system text").
		WithInstructionReader(strings.NewReader("text from reader")).
		WithText(llmberjack.RoleUser, "user text").
		WithTools(llmberjack.NewTool[Args]("thetool", "Tool to get nothing", llmberjack.Function(func(Args) (string, error) {
			return "OK", nil
		}))).
		WithTextReader(llmberjack.RoleUser, strings.NewReader("text from reader"))

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
	assert.Equal(t, llmberjack.FinishReasonStop, candidate.FinishReason)

	output, err := resp.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "The JSON response from the provider.", output.Reply)
}

func TestGoogleAiRequestWithThinking(t *testing.T) {
	defer gock.Off()

	httpClient := &http.Client{}
	provider, _ := aistudio.New(aistudio.WithBackend(genai.BackendVertexAI), aistudio.WithLocation("location"), aistudio.WithProject("project"))
	llm, _ := llmberjack.New(llmberjack.WithDefaultProvider(provider), llmberjack.WithHttpClient(httpClient))
	gock.InterceptClient(httpClient)

	tests := []struct {
		name            string
		thinking        *bool
		requestOptions  *aistudio.RequestOptions
		expectedMatcher func(body []byte) bool
	}{
		{
			name:           "Without requestOption",
			thinking:       nil,
			requestOptions: nil,
			expectedMatcher: func(body []byte) bool {
				// When no thinking config is provided, these fields should not be present
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				return true
			},
		},
		{
			name:     "With requestOption - only IncludeThoughts",
			thinking: nil,
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					IncludeThoughts: true,
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.EqualValues(t, true, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Bool())
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				return true
			},
		},
		{
			name:     "With requestOption - only Budget",
			thinking: nil,
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					Budget: lo.ToPtr(int(50)),
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.EqualValues(t, 50, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
		{
			name:     "With requestOption - both fields set",
			thinking: nil,
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					IncludeThoughts: true,
					Budget:          lo.ToPtr(int(100)),
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.EqualValues(t, true, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Bool())
				assert.EqualValues(t, 100, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
		{
			name: "With requestOption - Disable thinking",
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					Budget: lo.ToPtr(int(0)),
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.True(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				assert.EqualValues(t, 0, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
		{
			name:           "With Request - Disable thinking",
			thinking:       lo.ToPtr(false),
			requestOptions: nil,
			expectedMatcher: func(body []byte) bool {
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.True(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				assert.EqualValues(t, 0, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
		{
			name:     "With Request - Disable thinking with request option",
			thinking: lo.ToPtr(false),
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					Budget: lo.ToPtr(int(100)),
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.True(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				assert.EqualValues(t, 0, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
		{
			name:           "With Request - Enable thinking",
			thinking:       lo.ToPtr(true),
			requestOptions: nil,
			expectedMatcher: func(body []byte) bool {
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Exists())
				assert.False(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Exists())
				return true
			},
		},
		{
			name:     "With Request - Enable thinking with request option",
			thinking: lo.ToPtr(true),
			requestOptions: &aistudio.RequestOptions{
				Thinking: &aistudio.ThinkingConfig{
					IncludeThoughts: true,
					Budget:          lo.ToPtr(int(100)),
				},
			},
			expectedMatcher: func(body []byte) bool {
				assert.True(t, gjson.GetBytes(body, "generationConfig.thinkingConfig.includeThoughts").Bool())
				assert.EqualValues(t, 100, gjson.GetBytes(body, "generationConfig.thinkingConfig.thinkingBudget").Int())
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := llmberjack.NewUntypedRequest().
				WithModel("themodel").
				WithText(llmberjack.RoleUser, "user text")

			if tt.thinking != nil {
				req = req.WithThinking(*tt.thinking)
			}

			// Only add provider options if they exist
			if tt.requestOptions != nil {
				req = req.WithProviderOptions(*tt.requestOptions)
			}

			gock.New("https://location-aiplatform.googleapis.com").
				Post("/v1beta1/projects/project/locations/location/publishers/google/models/themodel:generateContent").
				AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
					body, _ := io.ReadAll(req.Body)
					return tt.expectedMatcher(body), nil
				}).
				Reply(http.StatusOK)

			_, err := req.Do(t.Context(), llm)
			assert.Nil(t, err)
			assert.False(t, gock.HasUnmatchedRequest())

			// Clean up gock for next subtest
			gock.Flush()
		})
	}
}

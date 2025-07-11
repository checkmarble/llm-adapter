package llmadapter_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
)

const openaiResponse = `
{
	"model": "themodel",
	"choices": [
	    {
			"index": 0,
			"message": {
		        "type": "message",
		        "role": "assistant",
		        "content": "{\"reply\":\"The JSON response from the provider.\"}"
			}
	    }
	]
}
`

func TestFullRequest(t *testing.T) {
	defer gock.Off()

	type Output struct {
		Reply string `json:"reply" jsonschema_description:"Write your response here"`
	}

	type Args struct {
		Name string `json:"name" jsonschema_description:"My name"`
	}

	reader := strings.NewReader("text from reader")
	provider, _ := openai.New()
	llm, _ := llmadapter.NewLlmAdapter(llmadapter.WithProvider(provider), llmadapter.WithApiKey("apikey"))

	req := llmadapter.NewRequest[Output]().
		WithModel("themodel").
		WithInstruction("system text").
		WithInstructionReader(reader).
		WithText(llmadapter.RoleUser, "user text").
		WithTools(llmadapter.NewTool[Args]("thetool", "Tool to get nothing", llmadapter.Function(func(Args) (string, error) {
			return "OK", nil
		}))).
		WithTextReader(llmadapter.RoleUser, reader)

	gock.New("https://api.openai.com").
		Post("/v1/chat/completions").
		MatchHeader("authorization", "Bearer apikey").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, _ := io.ReadAll(req.Body)
			matched := bytes.Equal(body, []byte(`{"messages":[{"content":[{"text":"system text","type":"text"}],"role":"system"},{"content":[{"text":"text from reader","type":"text"}],"role":"system"},{"content":[{"text":"user text","type":"text"}],"role":"user"},{"content":[{"text":"","type":"text"}],"role":"user"}],"model":"themodel","response_format":{"json_schema":{"name":"","strict":true,"description":"","schema":{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://github.com/checkmarble/marble-llm-adapter_test/output","properties":{"reply":{"type":"string","description":"Write your response here"}},"additionalProperties":false,"type":"object","required":["reply"]}},"type":"json_schema"},"tools":[{"function":{"name":"thetool","description":"Tool to get nothing","parameters":{"$id":"https://github.com/checkmarble/marble-llm-adapter_test/args","$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,"properties":{"name":{"description":"My name","type":"string"}},"required":["name"],"type":"object"}},"type":"function"}]}`))

			return matched, nil
		}).
		Reply(http.StatusOK).
		SetHeader("content-type", "application/json").BodyString(openaiResponse)

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

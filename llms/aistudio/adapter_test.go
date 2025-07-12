package aistudio

import (
	"strings"
	"testing"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestRequestAdapter(t *testing.T) {
	llm, _ := llmadapter.NewLlmAdapter()
	p, _ := New()

	t.Run("with model", func(t *testing.T) {
		req := llmadapter.NewUntypedRequest().
			WithModel("themodel")

		contents, cfg, err := p.adaptRequest(llm, req)

		assert.Nil(t, err)
		assert.Nil(t, cfg.SystemInstruction)
		assert.Len(t, contents, 0)
	})

	t.Run("with system prompts", func(t *testing.T) {
		req := llmadapter.NewUntypedRequest().
			WithModel("themodel").
			WithInstruction("system prompt", "system prompt 2").
			WithInstructionReader(strings.NewReader("system prompt 3"))

		contents, cfg, err := p.adaptRequest(llm, req)

		assert.Nil(t, err)
		assert.NotNil(t, cfg.SystemInstruction)
		assert.Len(t, cfg.SystemInstruction.Parts, 3)
		assert.Equal(t, "system prompt", cfg.SystemInstruction.Parts[0].Text)
		assert.Equal(t, "system prompt 2", cfg.SystemInstruction.Parts[1].Text)
		assert.Equal(t, "system prompt 3", cfg.SystemInstruction.Parts[2].Text)
		assert.Len(t, contents, 0)
	})

	t.Run("with user prompts", func(t *testing.T) {
		req := llmadapter.NewUntypedRequest().
			WithText(llmadapter.RoleUser, "user prompt", "user prompt 2").
			WithTextReader(llmadapter.RoleUser, strings.NewReader("user prompt 3"))

		contents, _, err := p.adaptRequest(llm, req)

		assert.Nil(t, err)
		assert.Len(t, contents, 2)
		assert.Len(t, contents[0].Parts, 2)
		assert.Equal(t, "user prompt", contents[0].Parts[0].Text)
		assert.Equal(t, "user prompt 2", contents[0].Parts[1].Text)
		assert.Len(t, contents[1].Parts, 1)
		assert.Equal(t, "user prompt 3", contents[1].Parts[0].Text)
	})

	t.Run("with tools", func(t *testing.T) {
		type Args1 struct {
			Number int `json:"number" jsonschema_description:"Number description"`
		}
		type Args2 struct {
			Text string `json:"text" jsonschema_description:"Text description"`
		}

		req := llmadapter.NewUntypedRequest().
			WithTools(
				llmadapter.NewTool[Args1]("toolname", "tooldesc", llmadapter.Function(func(a string) (string, error) {
					return "", nil
				})),
				llmadapter.NewTool[Args2]("toolname 2", "tooldesc 2", llmadapter.Function(func(a string) (string, error) {
					return "", nil
				})),
			)

		_, cfg, err := p.adaptRequest(llm, req)

		assert.Nil(t, err)
		assert.Len(t, cfg.Tools, 2)

		matchedTools := 0

		for _, tool := range cfg.Tools {
			matchedTools += 1

			assert.Len(t, tool.FunctionDeclarations, 1)

			schema, ok := tool.FunctionDeclarations[0].ParametersJsonSchema.(jsonschema.Schema)

			assert.True(t, ok)
			assert.Equal(t, "object", schema.Type)

			if tool.FunctionDeclarations[0].Name == "toolname" {
				assert.Equal(t, "tooldesc", tool.FunctionDeclarations[0].Description)
				assert.ElementsMatch(t, schema.Required, []string{"number"})
				assert.Equal(t, 1, schema.Properties.Len())
				assert.Equal(t, "integer", schema.Properties.Value("number").Type)
				assert.Equal(t, "Number description", schema.Properties.Value("number").Description)
			}
			if tool.FunctionDeclarations[0].Name == "toolname 2" {
				assert.Equal(t, "tooldesc 2", tool.FunctionDeclarations[0].Description)
				assert.ElementsMatch(t, schema.Required, []string{"text"})
				assert.Equal(t, 1, schema.Properties.Len())
				assert.Equal(t, "string", schema.Properties.Value("text").Type)
				assert.Equal(t, "Text description", schema.Properties.Value("text").Description)
			}
		}

		assert.Equal(t, 2, matchedTools)
	})

	t.Run("with response format", func(t *testing.T) {
		type Format struct {
			Text   string `json:"text" jsonschema_description:"Text description"`
			Number int    `json:"number" jsonschema_description:"Number description"`
		}

		req := llmadapter.NewRequest[Format]()

		_, cfg, err := p.adaptRequest(llm, req)

		assert.Nil(t, err)
		assert.NotNil(t, cfg.ResponseJsonSchema)

		schema, ok := cfg.ResponseJsonSchema.(jsonschema.Schema)

		assert.True(t, ok)
		assert.Equal(t, "object", schema.Type)
		assert.ElementsMatch(t, schema.Required, []string{"text", "number"})
		assert.Equal(t, 2, schema.Properties.Len())
		assert.Equal(t, "string", schema.Properties.Value("text").Type)
		assert.Equal(t, "Text description", schema.Properties.Value("text").Description)
		assert.Equal(t, "integer", schema.Properties.Value("number").Type)
		assert.Equal(t, "Number description", schema.Properties.Value("number").Description)
	})
}

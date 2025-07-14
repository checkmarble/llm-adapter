package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSchema(t *testing.T) {
	type Type struct {
		Text     string `json:"text" jsonschema_description:"Text description"`
		TextEnum string `json:"text_enum" jsonschema_description:"Text enum description" jsonschema:"enum=one,enum=two"`
		Number   int    `json:"number" jsonschema_description:"Number description"`
		Array    []int  `json:"array" jsonschema_description:"Array description"`
		Object   struct {
			Number int `json:"number" jsonschema_description:"Number description"`
		} `json:"object" jsonschema_description:"Object description"`
		ArrayOfObjects []struct {
			Number int `json:"number" jsonschema_description:"Number description"`
		} `json:"array_of_objects" jsonschema_description:"Array of objects description"`
	}

	schema := GenerateSchema[Type]()

	assert.Equal(t, "object", schema.Type)
	assert.ElementsMatch(t, schema.Required, []string{"text", "text_enum", "number", "array", "object", "array_of_objects"})

	assert.Equal(t, "string", schema.Properties.Value("text").Type)
	assert.Equal(t, "Text description", schema.Properties.Value("text").Description)

	assert.Equal(t, "string", schema.Properties.Value("text_enum").Type)
	assert.Equal(t, "Text enum description", schema.Properties.Value("text_enum").Description)
	assert.ElementsMatch(t, schema.Properties.Value("text_enum").Enum, []any{"one", "two"})

	assert.Equal(t, "integer", schema.Properties.Value("number").Type)
	assert.Equal(t, "Number description", schema.Properties.Value("number").Description)

	assert.Equal(t, "array", schema.Properties.Value("array").Type)
	assert.Equal(t, "Array description", schema.Properties.Value("array").Description)
	assert.Equal(t, "integer", schema.Properties.Value("array").Items.Type)

	assert.Equal(t, "object", schema.Properties.Value("object").Type)
	assert.Equal(t, "Object description", schema.Properties.Value("object").Description)
	assert.ElementsMatch(t, schema.Properties.Value("object").Required, []string{"number"})
	assert.Equal(t, "integer", schema.Properties.Value("object").Properties.Value("number").Type)

	assert.Equal(t, "array", schema.Properties.Value("array_of_objects").Type)
	assert.Equal(t, "Array of objects description", schema.Properties.Value("array_of_objects").Description)
	assert.Equal(t, "object", schema.Properties.Value("array_of_objects").Items.Type)
	assert.ElementsMatch(t, schema.Properties.Value("array_of_objects").Items.Required, []string{"number"})
	assert.Equal(t, "integer", schema.Properties.Value("array_of_objects").Items.Properties.Value("number").Type)
}

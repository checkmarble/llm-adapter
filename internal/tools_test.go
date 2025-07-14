package internal

import (
	"fmt"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewTool(t *testing.T) {
	type Args struct {
		Number int `json:"number" jsonschema_description:"Number description"`
	}

	tool := NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args) (string, error) {
			return "", nil
		},
	})

	assert.Equal(t, "name", tool.Name)
	assert.Equal(t, "desc", tool.Description)

	assert.Equal(t, "integer", tool.Parameters.Properties.Value("number").Type)
	assert.Equal(t, "Number description", tool.Parameters.Properties.Value("number").Description)

	assert.IsType(t, Args{}, tool.input)
	assert.IsType(t, func(Args) (string, error) { return "", nil }, tool.function.Inner)
}

func TestCallTool(t *testing.T) {
	type Args struct {
		Number int `json:"number" jsonschema_description:"Number description"`
	}

	called := 0

	tool := NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args) (string, error) {
			called += args.Number

			return "called", nil
		},
	})

	output, err := tool.Call([]byte(`{"number":42}`))

	assert.Nil(t, err)
	assert.Equal(t, "called", output)
	assert.Equal(t, 42, called)
}

func TestCallToolWrongArguments(t *testing.T) {
	type Args struct {
		Number int `json:"number" jsonschema_description:"Number description"`
	}

	called := 0

	tool := NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args, other int) (string, error) {
			called += args.Number

			return "called", nil
		},
	})

	_, err := tool.Call([]byte(`{"number":42}`))

	assert.NotNil(t, err)
	assert.Equal(t, 0, called)

	tool = NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args string) (string, error) {
			called += 1

			return "called", nil
		},
	})

	_, err = tool.Call([]byte(`{"number":42}`))

	fmt.Println(err)

	assert.NotNil(t, err)
	assert.Equal(t, 0, called)

	tool = NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args) error {
			called += args.Number

			return nil
		},
	})

	_, err = tool.Call([]byte(`{"number":42}`))

	assert.NotNil(t, err)
	assert.Equal(t, 0, called)
}

func TestCallToolInvalidJson(t *testing.T) {
	type Args struct {
		Number int `json:"number" jsonschema_description:"Number description"`
	}

	called := 0

	tool := NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args) (string, error) {
			called += args.Number

			return "called", nil
		},
	})

	_, err := tool.Call([]byte(`{"number":"ok"}`))

	assert.NotNil(t, err)
	assert.Equal(t, 0, called)
}

func TestCallToolReturnsError(t *testing.T) {
	type Args struct {
		Number int `json:"number" jsonschema_description:"Number description"`
	}

	called := 0

	tool := NewTool[Args]("name", "desc", FunctionBody{
		Inner: func(args Args) (string, error) {
			called += args.Number

			return "called", errors.New("inner error")
		},
	})

	_, err := tool.Call([]byte(`{"number":42}`))

	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "inner error")
	assert.Equal(t, 42, called)
}

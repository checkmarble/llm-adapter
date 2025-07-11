package llmadapter

import (
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
)

// Tool is a tool function definition.
type Tool struct {
	Name        string
	Description string
	Parameters  jsonschema.Schema

	function functionBody
	input    any
}

type functionBody struct {
	inner any
}

// Function is a wrapper for the code executed in a tool.
//
// It is generic in I, which is a type containing the tool arguments. It
// follows the same idioms as a response schema.
func Function[I any](f func(I) (string, error)) functionBody {
	return functionBody{any(f)}
}

// NewTool creates a new tool.
//
// It is generic in the type of the tool arguments, and takes the tool name
// and description.
//
// The function body should be wrapped in `Function`.
func NewTool[T any](name, description string, fn functionBody) Tool {
	return Tool{
		Name:        name,
		Description: description,
		Parameters:  generateSchema[T]("", "").Schema,
		input:       *new(T),
		function:    fn,
	}
}

func (t Tool) call(paramsJson []byte) (string, error) {
	params := reflect.New(reflect.TypeOf(t.input)).Interface()

	if err := json.Unmarshal(paramsJson, &params); err != nil {
		return "", err
	}

	fn := reflect.ValueOf(t.function.inner)
	args := []reflect.Value{reflect.ValueOf(params).Elem()}
	rets := fn.Call(args)

	if len(rets) != 2 {
		panic("tool functions should return (string, error)")
	}

	if !rets[1].IsNil() {
		return "", rets[1].Interface().(error)
	}

	output, ok := rets[0].Interface().(string)

	if !ok {
		panic("tool function should return (string, error)")
	}

	return output, nil
}

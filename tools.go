package llmadapter

import (
	"encoding/json"
	"reflect"

	"github.com/invopop/jsonschema"
)

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

func Function[I any](f func(I) (string, error)) functionBody {
	return functionBody{any(f)}
}

func NewTool[T any](name, description string, fn functionBody) Tool {
	return Tool{
		Name:        name,
		Description: description,
		Parameters:  GenerateSchema[T]("", "").Schema,
		input:       *new(T),
		function:    fn,
	}
}

func (t Tool) Call(paramsJson []byte) (string, error) {
	paramsType := reflect.TypeOf(t.input)
	params := reflect.New(paramsType).Interface()

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

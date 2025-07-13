package internal

import (
	"encoding/json"
	"reflect"

	"github.com/cockroachdb/errors"
	"github.com/invopop/jsonschema"
)

// Tool is a tool function definition.
//
// This type, as well as this whole file, is in the internal package so its
// internals cannot be manually crafted, and we can guarantee a semblance of
// type safety.
type Tool struct {
	Name        string
	Description string
	Parameters  jsonschema.Schema

	// input is the erased type the function arguments
	input any
	// function is the actual function pointer
	function FunctionBody
}

// NewTool is only called by the public-facing NewTool function.
func NewTool[A any](name, description string, fn FunctionBody) Tool {
	return Tool{
		Name:        name,
		Description: description,
		Parameters:  GenerateSchema[A](),
		input:       *new(A),
		function:    fn,
	}
}

// FunctionBody is a wrapper around the tool function pointer.
//
// It is private so the only way to create it is through the
// llmadapter.Function[A]() function, which ensures the argument is of the shape
// `func(a) (string, error)`.
type FunctionBody struct {
	Inner any
}

// call resolves the tool function and executes it.
//
// It does some reflection dark magic from the recorded type-erased values on
// Tool[A] to deserialize the JSON-encoded arguments from the provider into A,
// retrieve the function pointer, check its shape (number and types of arguments
// and return values), and call it.
//
// We are being a bit overly cautious here, some of the checks are on supposed
// invariants, but better safe than sorry.
func (t Tool) Call(paramsJson []byte) (string, error) {
	// t.input is the type-erased recorded type of the function argument
	argType := reflect.TypeOf(t.input)
	params := reflect.New(argType).Interface()

	if err := json.Unmarshal(paramsJson, &params); err != nil {
		return "", err
	}

	// fn is our function pointer
	fn := reflect.ValueOf(t.function.Inner)

	// This should not be necessary because the only way to build a FunctionBody ensures the callback has one argument.
	if fn.Type().NumIn() != 1 {
		return "", errors.Newf("tool '%s' should take one argument, not %d", t.Name, fn.Type().NumIn())
	}
	// This is important, we cannot enforce the function argument type, so we need to check it to prevent panics.
	if fn.Type().In(0) != argType {
		return "", errors.Newf("tool '%s' should take an argument of type %s, not %s", t.Name, argType.Name(), fn.Type().In(0).Name())
	}

	args := []reflect.Value{reflect.ValueOf(params).Elem()}
	rets := fn.Call(args)

	// Once again, this should still be an invariant of the only function in the public API can build FunctionBody.
	if len(rets) != 2 {
		panic("tool functions should return (string, error)")
	}

	// Code path when the function returns an error
	if !rets[1].IsNil() {
		return "", rets[1].Interface().(error)
	}

	// Otherwise, we should have a string here.
	output, ok := rets[0].Interface().(string)

	if !ok {
		panic("tool function should return (string, error)")
	}

	return output, nil
}

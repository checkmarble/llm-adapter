package llmberjack

import (
	"github.com/checkmarble/llmberjack/internal"
)

// Function is a wrapper for the code executed in a tool.
//
// It is generic in A, which is a type containing the tool arguments. It
// follows the same idioms as a response schema.
func Function[A any](f func(args A) (string, error)) internal.FunctionBody {
	return internal.FunctionBody{Inner: any(f)}
}

// NewTool creates a new tool.
//
// It is generic in the type of the tool arguments, and takes the tool name
// and description.
//
// The function body should be wrapped in `Function`.
func NewTool[A any](name, description string, fn internal.FunctionBody) internal.Tool {
	return internal.NewTool[A](name, description, fn)
}

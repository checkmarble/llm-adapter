package llmadapter

import (
	"encoding/json"
	"io"
)

var (
	// Serializers is a global object that hold singleton of library-provided
	// serializers
	Serializers = struct {
		Json jsonSerializer
	}{
		Json: jsonSerializer{},
	}
)

type Serializer interface {
	Serialize(input any, output io.Writer) error
}

type jsonSerializer struct{}

func (jsonSerializer) Serialize(input any, output io.Writer) error {
	return json.NewEncoder(output).Encode(input)
}

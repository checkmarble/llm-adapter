package llmadapter

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"reflect"

	"github.com/cockroachdb/errors"
)

var (
	// Serializers is a global object that hold singleton of library-provided
	// serializers
	Serializers = struct {
		Json jsonSerializer
		Csv  csvSerializer
	}{
		Json: jsonSerializer{},
		Csv:  csvSerializer{},
	}
)

type Serializer interface {
	Serialize(input any, output io.Writer) error
}

type jsonSerializer struct{}

func (jsonSerializer) Serialize(input any, output io.Writer) error {
	return json.NewEncoder(output).Encode(input)
}

type csvSerializer struct{}

func (csvSerializer) Serialize(input any, output io.Writer) error {
	t := reflect.TypeOf(input)

	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Slice || t.Elem().Elem().Kind() != reflect.String {
		return errors.New("CSV serializer accepts a [][]string")
	}

	enc := csv.NewWriter(output)

	return enc.WriteAll(input.([][]string))
}

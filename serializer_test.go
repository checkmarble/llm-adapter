package llmberjack

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonSerializer(t *testing.T) {
	type j struct {
		Output string `json:"output"`
	}

	data := j{"Hello, world!"}
	req := NewUntypedRequest().WithSerializable(RoleUser, Serializers.Json, data)

	assert.Nil(t, req.err)
	assert.Len(t, req.Messages, 1)

	out, err := io.ReadAll(req.Messages[0].Parts[0])

	assert.Nil(t, err)
	assert.JSONEq(t, `{"output":"Hello, world!"}`, string(out))
}

func TestCsvSerializer(t *testing.T) {
	data := [][]string{
		{"one", "two"},
		{"three", "four"},
		{"five", "six"},
	}

	req := NewUntypedRequest().WithSerializable(RoleUser, Serializers.Csv, data)

	assert.Nil(t, req.err)
	assert.Len(t, req.Messages, 1)

	out, err := io.ReadAll(req.Messages[0].Parts[0])

	assert.Nil(t, err)
	assert.Equal(t, "one,two\nthree,four\nfive,six\n", string(out))
}

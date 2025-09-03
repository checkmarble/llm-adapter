package llmberjack

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	httpClient := &http.Client{}

	llm, err := New()

	assert.Nil(t, err)
	assert.Nil(t, llm.HttpClient())

	llm, err = New(WithDefaultModel("themodel"))

	assert.Nil(t, err)
	assert.Equal(t, "themodel", llm.DefaultModel())

	llm, err = New(WithHttpClient(httpClient))

	assert.Nil(t, err)
	assert.Equal(t, httpClient, llm.HttpClient())
}

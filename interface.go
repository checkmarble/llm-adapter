package llmadapter

import (
	"context"
)

type Llm interface {
	Init(llm Adapter) error
	ResetContext()
	ChatCompletion(context.Context, Adapter, LlmRequester) (*Response, error)
}

type Adapter interface {
	DefaultModel() string
	ApiKey() string
	SaveContext() bool
}

func (llm LlmAdapter) DefaultModel() string {
	return llm.defaultModel
}

func (llm LlmAdapter) ApiKey() string {
	return llm.apiKey
}

func (llm LlmAdapter) SaveContext() bool {
	return llm.saveContext
}

type LlmRequester interface {
	ToRequest() innerRequest
}

func (r Request[T]) ToRequest() innerRequest {
	return r.innerRequest
}

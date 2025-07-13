package llmadapter

import (
	"context"
	"reflect"

	"github.com/checkmarble/marble-llm-adapter/internal"
)

type Llm interface {
	Init(llm internal.Adapter) error
	ResetContext()
	ChatCompletion(context.Context, internal.Adapter, LlmRequester) (*Response, error)
	RequestOptionsType() reflect.Type
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
	ProviderRequestOptions(provider Llm) internal.ProviderRequestOptions
}

func (r Request[T]) ToRequest() innerRequest {
	return r.innerRequest
}

func (r Request[T]) ProviderRequestOptions(provider Llm) internal.ProviderRequestOptions {
	var providerOpts internal.ProviderRequestOptions

	if opts, ok := r.ProviderOptions[provider.RequestOptionsType()]; ok {
		providerOpts = opts
	}

	return providerOpts
}

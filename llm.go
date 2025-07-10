package llmadapter

import "context"

type Llm interface {
	Init(llm LlmAdapter) error
	ResetContext()
	ChatCompletions(context.Context, *LlmAdapter, InnerRequest) (*Response, error)
}

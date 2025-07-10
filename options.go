package llmadapter

type llmOption func(*LlmAdapter)

func WithProvider(provider Llm) llmOption {
	return func(llm *LlmAdapter) {
		llm.provider = provider
	}
}

func WithDefaultModel(model string) llmOption {
	return func(llm *LlmAdapter) {
		llm.DefaultModel = model
	}
}

func WithApiKey(key string) llmOption {
	return func(llm *LlmAdapter) {
		llm.ApiKey = key
	}
}

func WithSaveContext() llmOption {
	return func(llm *LlmAdapter) {
		llm.SaveContext = true
	}
}
